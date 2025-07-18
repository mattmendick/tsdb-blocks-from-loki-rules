#!/bin/bash

# Script to generate TSDB blocks for 30 days of Loki data
# Runs one day at a time to stay within Loki's 11,000 point limit

LOKI_URL="https://logs-prod-us-central1.grafana.net"
USERNAME="grafana cloud id"
PASSWORD="grafana cloud token"
RULES_FILE="rules.yaml"
OUTPUT_DIR="./data-30days"
BLOCK_DURATION="24h"  # Configurable block duration: 2h, 6h, 24h, 168h (7d), etc.

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "Starting 30-day data generation..."
echo "Output directory: $OUTPUT_DIR"
echo "Rules file: $RULES_FILE"
echo "Block duration: $BLOCK_DURATION"
echo ""

# Function to run the tool for a specific day
run_for_day() {
    local start_date=$1
    local end_date=$2
    local day_label=$3
    
    echo "Processing $day_label..."
    echo "  Start: $start_date"
    echo "  End: $end_date"
    
    ./loki-tsdb-tool \
        --rules="$RULES_FILE" \
        --loki.url="$LOKI_URL" \
        --loki.username="$USERNAME" \
        --loki.password="$PASSWORD" \
        --start="$start_date" \
        --end="$end_date" \
        --output.dir="$OUTPUT_DIR" \
        --block.duration="$BLOCK_DURATION"
    
    if [ $? -eq 0 ]; then
        echo "  ✅ Completed $day_label"
    else
        echo "  ❌ Failed $day_label"
        exit 1
    fi
    echo ""
}

# Generate blocks for each day
# June 16-30 (15 days)
for day in {16..30}; do
    start_date="2025-06-${day}T00:00:00Z"
    end_date="2025-06-${day}T23:59:59Z"
    day_label="June $day, 2025"
    run_for_day "$start_date" "$end_date" "$day_label"
done

# July 1-14 (14 days) - properly zero-padded days
for day in {1..14}; do
    padded_day=$(printf "%02d" $day)
    start_date="2025-07-${padded_day}T00:00:00Z"
    end_date="2025-07-${padded_day}T23:59:59Z"
    day_label="July $day, 2025"
    run_for_day "$start_date" "$end_date" "$day_label"
done

# July 15 (partial day, ending at 16:27:00)
start_date="2025-07-15T00:00:00Z"
end_date="2025-07-15T16:27:00Z"
day_label="July 15, 2025 (partial)"
run_for_day "$start_date" "$end_date" "$day_label"

echo "🎉 All 30 days completed successfully!"
echo "Total blocks generated in: $OUTPUT_DIR"
echo "Block duration used: $BLOCK_DURATION"
echo ""
echo "You can now use promtool to list the blocks:"
echo "promtool tsdb list $OUTPUT_DIR" 
