# Helpful scripts

This directory contains a number of simple, but helpful scripts described below.

### apply-all.sh

This script applies all the infrastructure necessary for monitoring the tests in one go.
Namely, it installs Prometheus, Grafana.

### clear-prometheus.sh

This script resets prometheus data by deleting all running prometheus instances and
letting them get re-created.  This is useful for resetting state between tests.

### delete-all.sh

The counterpart to `apply-all.sh` - this deletes all this infrastructure from the cluster.

### gather-client-logs.sh

Gathers all the logs from clients pods created by the test using `kubectl logs`.

### gather-felix-logs.sh

Gathers `calico/node` logs from each instance running on the cluster using
`kubectl logs`.

### gather-logs-the-hard-way.sh

Scrapes logs via gcloud ssh - can be used when kubectl isn't working.

### gather-prometheus.sh

Generates `.tar` files of the data from each running prometheus instance and
stores it locally so it can be viewed after the cluster has been torn down.

### reapply-grafana-dash.sh

Re-applies grafana dashboards in the case where the configuration is lost
or failed to apply.

### lb2nodeport.sh

Changes LoadBalancer services into Nodeport services, for when your cluster
doesn't have the cloud integration installed for creating LBs.

### prometheus_grafana_stack
Contains scripts that will setup the kubernetes, prometheus, and grafana monitoring stack on your cluster using helm. It comes pre-loaded with several useful kubernetes cluster, deployment, and pod level dashboards.