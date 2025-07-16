# tsdb-blocks-from-loki-rules

This tool creates Prometheus TSDB blocks from historical Loki query results, using a rules YAML file (similar to Prometheus recording rules).

## Usage

```
go build -o loki-tsdb-tool
./loki-tsdb-tool \
  --rules=rules.yaml \
  --loki.url=https://your-loki-endpoint/loki/api/v1/query_range \
  --output.dir=./data \
  --start=2024-05-01T00:00:00Z \
  --end=2024-05-02T00:00:00Z \
  --loki.username=YOUR_USERNAME \
  --loki.password=YOUR_PASSWORD
```

- `--rules`: Path to your rules YAML file (see below for format)
- `--loki.url`: Loki base URL (should end with `/loki/api/v1/query_range` or just the base, which will be appended)
- `--output.dir`: Output directory for TSDB blocks
- `--start`, `--end`: Time range to backfill (RFC3339 or Unix timestamp)
- `--loki.username`, `--loki.password`: Basic auth credentials for Loki (optional)

## Example rules.yaml

```yaml
groups:
  - name: loki-metrics
    step: "60s"  # Default step for all rules in this group
    rules:
      - name: http_requests_total
        query: sum by (job) (rate({app="myapp"} |~ "level=info" [1m]))
        step: "60s"  # Override step for this specific rule
        labels:
          source: loki
      - name: error_count
        query: sum(count_over_time({app="myapp"} |~ "level=error" [1m]))
        labels:
          severity: error
```

## Step Configuration

The step size can be configured at three levels (in order of precedence):
1. **Rule level**: `step: "60s"` in individual rules (highest priority)
2. **Group level**: `step: "60s"` in group definition (default for all rules in group)
3. **CLI level**: `--step=60s` command line flag (fallback default)

## Notes
- The tool will create TSDB blocks in the output directory, one per block duration (default 2h).
- You can use `promtool tsdb dump` or Prometheus itself to inspect/import the blocks.
- Loki credentials are optional, but required for secured endpoints (e.g., Grafana Cloud Loki). 
