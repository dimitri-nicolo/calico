#!/bin/sh
set -e -o pipefail

# Location of readiness file which contains elasticsearch metrics
readiness_destfile="/tmp/readiness_metrics"
output=$(curl -s "http://localhost:24220/api/plugins.json")

# Get the current metrics of elasticsearch
retry_count_current=$(echo $output | jq --raw-output '.plugins[] | select(.config.index_name) | select(.config.index_name | test("tigera_secure_ee_flows")) | .retry_count' | awk '{SUM += $0 } END { print SUM }');
buffer_length_current=$(echo $output | jq --raw-output '.plugins[] | select(.config.index_name) | select(.config.index_name | test("tigera_secure_ee_flows")) | .buffer_total_queued_size' | awk '{SUM += $0 } END { print SUM }');

# Special case: If flow logs has been turned off, then skip the check and return ready 
if [ -z ${retry_count_current} ]; then
    echo "Flow logs to ES is disabled, skip the retry/buffer check"
    exit 0
fi

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
