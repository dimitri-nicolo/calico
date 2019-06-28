#!/bin/sh
set -e

# set up any environment variables necessary for our liveness check to run properly
source /bin/splunk-environment.sh
source /bin/sumo-environment.sh

/usr/bin/fluentd -c /fluentd/etc/fluent.conf -p /fluentd/plugins --dry-run && curl -s http://localhost:24220/api/plugins.json

