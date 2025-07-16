package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// QueryLokiRange queries Loki's /loki/api/v1/query_range and returns MatrixSample
func QueryLokiRange(lokiURL, query string, start, end time.Time, step, username, password string) (MatrixSample, error) {
	stepDur, err := time.ParseDuration(step)
	if err != nil {
		return nil, fmt.Errorf("invalid step: %w", err)
	}

	u, err := url.Parse(lokiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid loki url: %w", err)
	}
	if !strings.HasSuffix(u.Path, "/loki/api/v1/query_range") {
		u.Path = strings.TrimRight(u.Path, "/") + "/loki/api/v1/query_range"
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	q.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	q.Set("step", strconv.FormatFloat(stepDur.Seconds(), 'f', -1, 64))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki http %d: %s", resp.StatusCode, string(b))
	}

	var lokiResp lokiQueryRangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("decode loki response: %w", err)
	}
	if lokiResp.Status != "success" {
		return nil, errors.New("loki query failed: " + lokiResp.Status)
	}
	if lokiResp.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("unexpected result type: %s", lokiResp.Data.ResultType)
	}

	var matrix MatrixSample
	for _, s := range lokiResp.Data.Result {
		labels := make(map[string]string)
		for k, v := range s.Metric {
			labels[k] = v
		}
		var samples []Sample
		for _, arr := range s.Values {
			if len(arr) != 2 {
				continue
			}

			// Handle timestamp (could be float64 or string)
			var ts float64
			switch v := arr[0].(type) {
			case float64:
				ts = v
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					ts = parsed
				} else {
					continue
				}
			default:
				continue
			}

			// Handle value (could be float64 or string)
			var val float64
			switch v := arr[1].(type) {
			case float64:
				val = v
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					val = parsed
				} else {
					continue
				}
			default:
				continue
			}

			samples = append(samples, Sample{
				Timestamp: int64(ts * 1000), // Loki returns seconds, Prometheus expects ms
				Value:     val,
			})
		}
		matrix = append(matrix, TimeSeries{
			Labels:  labels,
			Samples: samples,
		})
	}
	return matrix, nil
}

type lokiQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string           `json:"resultType"`
		Result     []lokiMatrixElem `json:"result"`
	} `json:"data"`
}

type lokiMatrixElem struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}
