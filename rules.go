package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/tsdb"
)

type Rule struct {
	Name   string            `yaml:"name"`
	Query  string            `yaml:"query"`
	Step   string            `yaml:"step,omitempty"` // Optional step override
	Labels map[string]string `yaml:"labels"`
}

type RulesFile struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Step  string `yaml:"step,omitempty"` // Default step for group
		Rules []Rule `yaml:"rules"`
	} `yaml:"groups"`
}

func Run(rulesFile, lokiURL, outputDir, start, end, step, username, password string) error {
	// Parse times
	stime, err := parseTime(start)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	var etime time.Time
	if end == "" {
		etime = time.Now().UTC().Add(-3 * time.Hour)
	} else {
		etime, err = parseTime(end)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
	}
	if !stime.Before(etime) {
		return errors.New("start time must be before end time")
	}

	// Parse rules YAML
	f, err := os.Open(rulesFile)
	if err != nil {
		return fmt.Errorf("open rules file: %w", err)
	}
	defer f.Close()
	var rf RulesFile
	if err := yaml.NewDecoder(f).Decode(&rf); err != nil {
		return fmt.Errorf("parse rules yaml: %w", err)
	}

	// For each rule, query Loki and write blocks
	for _, group := range rf.Groups {
		groupStep := group.Step
		if groupStep == "" {
			groupStep = step // Fall back to CLI step if not specified in group
		}

		for _, rule := range group.Rules {
			fmt.Printf("Processing rule: %s\n", rule.Name)

			// Use rule-specific step, then group step, then CLI step
			ruleStep := rule.Step
			if ruleStep == "" {
				ruleStep = groupStep
			}

			matrix, err := QueryLokiRange(lokiURL, rule.Query, stime, etime, ruleStep, username, password)
			if err != nil {
				return fmt.Errorf("loki query failed for rule %s: %w", rule.Name, err)
			}
			if err := writeTSDBBlocks(outputDir, rule, matrix, stime, etime); err != nil {
				return fmt.Errorf("write TSDB blocks for rule %s: %w", rule.Name, err)
			}
		}
	}
	return nil
}

// parseTime parses RFC3339 or Unix timestamp
func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid time: %q", s)
}

// MatrixSample is a Prometheus-style matrix: []TimeSeries
// Each TimeSeries is a set of labels and a list of (timestamp, value) pairs
// This matches Prometheus's model.Matrix

type MatrixSample []TimeSeries

type TimeSeries struct {
	Labels  map[string]string
	Samples []Sample
}

type Sample struct {
	Timestamp int64
	Value     float64
}

// writeTSDBBlocks writes the matrix samples to TSDB blocks using tsdb.BlockWriter
func writeTSDBBlocks(outputDir string, rule Rule, matrix MatrixSample, stime, etime time.Time) error {
	const blockDuration = 2 * time.Hour // Prometheus default block duration
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mint := stime.Unix() * 1000 // ms
	maxt := etime.Unix() * 1000 // ms

	for blockStart := mint - (mint % int64(blockDuration/time.Millisecond)); blockStart < maxt; blockStart += int64(blockDuration / time.Millisecond) {
		blockEnd := blockStart + int64(blockDuration/time.Millisecond) - 1
		if blockEnd > maxt {
			blockEnd = maxt
		}

		w, err := tsdb.NewBlockWriter(logger, outputDir, 2*int64(blockDuration/time.Millisecond))
		if err != nil {
			return fmt.Errorf("new block writer: %w", err)
		}
		app := w.Appender(ctx)
		seriesCount := 0
		for _, ts := range matrix {
			labelsMap := make(map[string]string)
			for k, v := range ts.Labels {
				labelsMap[k] = v
			}
			for k, v := range rule.Labels {
				labelsMap[k] = v // rule labels override
			}
			labelsMap["__name__"] = rule.Name

			lbls := labels.FromMap(labelsMap)

			for _, s := range ts.Samples {
				if s.Timestamp < blockStart || s.Timestamp > blockEnd {
					continue
				}
				_, err := app.Append(0, lbls, s.Timestamp, s.Value)
				if err != nil {
					return fmt.Errorf("append: %w", err)
				}
				seriesCount++
			}
		}
		if err := app.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		if _, err := w.Flush(ctx); err != nil {
			return fmt.Errorf("flush: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}
		if seriesCount > 0 {
			fmt.Printf("Wrote block: [%d, %d] with %d samples\n", blockStart, blockEnd, seriesCount)
		}
	}
	return nil
}
