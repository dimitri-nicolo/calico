## Intrusion Detection jobs and software for Calico Enterprise

### Contents
  * `install/` - Installer for Elastic Stack (ML jobs, watches, dashboards, etc.)
  * `cmd/controller/` - Controller for intrusion detection jobs
  * `pkg/controller/` - Go packages for the controller for intrusion detection jobs
  * `test/` - Test applications, command, etc.

### Migrating Kibana Dashboards
When Kibana and Elasticsearch versions are upgraded the dashboard json in the `install/data` directory may need to be upgraded to
be compatible with the new version. Luckily, when Kibana and Elasticsearch are upgraded with dashboards already loaded
the Dashboards will be upgraded internally. All we need to do is export the upgrade version from Elasticsearch and update
the json files in `install/data`.

The process for upgrading the dashboards is as follows:
  1. Create a K8s cluster with Calico Enterprise installed with the previous version of Elasticsearch and Kibana.
  2. Upgrade Elasticsearch and Kibana.
  3. What for Elasticsearch and Kibana to be functional.
  4. Port forward the Tigera Manager.
  5. Run `make migrate-dashboards` in the `install` directory (this exports the dashboards from Kibana and updates the json for each dashboard).
  6. Create a PR for the changes created in step 5.
