#!/bin/bash

echo "Testing July date formatting:"
for day in {01..05}; do
    start_date="2025-07-${day}T00:00:00Z"
    end_date="2025-07-${day}T23:59:59Z"
    day_label="July ${day#0}, 2025"
    echo "Day: $day"
    echo "  Start: $start_date"
    echo "  End: $end_date"
    echo "  Label: $day_label"
    echo ""
done 
