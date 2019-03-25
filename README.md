[![Build Status](https://semaphoreci.com/api/v1/projects/f873f23a-d7c5-442d-aa56-2968f74e992f/2591090/shields_badge.svg)](https://semaphoreci.com/calico/compliance)

# Tigera compliance

Components related to the compliance dashboard and reporting feature of TSEE.

Contents
  * config-snapshotter/ - Periodic snapshotter required for configuration replays using audit logs
  * report-generator/ - The job that generates a report
  * report-generator-scheduler/ - The scheduler used for creating single-run report-generator jobs
