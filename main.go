package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		rulesFile     = flag.String("rules", "", "YAML file with Loki rules")
		lokiURL       = flag.String("loki.url", "", "Loki base URL")
		outputDir     = flag.String("output.dir", "data", "Output directory for TSDB blocks")
		start         = flag.String("start", "", "Start time (RFC3339 or Unix timestamp)")
		end           = flag.String("end", "", "End time (RFC3339 or Unix timestamp)")
		step          = flag.String("step", "60s", "Default query step size (duration) - can be overridden in rules file")
		blockDuration = flag.String("block.duration", "2h", "Block duration (e.g., 2h, 6h, 1d, 7d)")
		username      = flag.String("loki.username", "", "Loki basic auth username")
		password      = flag.String("loki.password", "", "Loki basic auth password")
	)
	flag.Parse()

	if *rulesFile == "" || *lokiURL == "" || *start == "" {
		fmt.Fprintln(os.Stderr, "--rules, --loki.url, and --start are required")
		os.Exit(1)
	}

	err := Run(*rulesFile, *lokiURL, *outputDir, *start, *end, *step, *blockDuration, *username, *password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
