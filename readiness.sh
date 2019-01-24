#!/bin/sh

# Location of readiness file which contains elasticsearch metrics
readiness_destfile="/tmp/readiness_metrics"

# Get the current metrics of elasticsearch
retry_count_current=$(curl -s http://localhost:24220/api/plugins.json | jq --raw-output '.plugins[] | select(.config.index_name) | select(.config.index_name | test("tigera_secure_ee_flows")) | .retry_count' | awk '{SUM += $0 } END { print SUM }');
buffer_length_current=$(curl -s http://localhost:24220/api/plugins.json | jq --raw-output '.plugins[] | select(.config.index_name) | select(.config.index_name | test("tigera_secure_ee_flows")) | .buffer_total_queued_size' | awk '{SUM += $0 } END { print SUM }');

# Check if earlier metrics present
if [ ! -f "$readiness_destfile" ]
then
    echo "$retry_count_current $buffer_length_current" > "$readiness_destfile"
	exit 0
fi

# Get the earlier metrics from readiness file
data=`cat $readiness_destfile`
retry_count_prev=$(echo $data | cut -d' ' -f1)
buffer_length_prev=$(echo $data | cut -d' ' -f2)

# Write new metrics in readiness file for further reference
echo "$retry_count_current $buffer_length_current" > "$readiness_destfile"

# Compare the new metrics with the earlier metrics of elasticsearch
if [[ $retry_count_current -gt $retry_count_prev || $buffer_length_current -gt $buffer_length_prev ]];
then
    exit 1;
fi
exit 0;
