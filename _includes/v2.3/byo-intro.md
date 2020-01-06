{% if include.orch != "openshift" %}
  {% assign path = "kubernetes" %}
  {% assign name = "Kubernetes" %}
{% else %}
  {% assign path = "openshift" %}
  {% assign name = "OpenShift" %}
{% endif %}
## About using your own Elasticsearch

For production clusters, you must use your own Elasticsearch cluster. We bundle an Elasticsearch
operator for quickstart and demonstration purposes, but it is not suitable for production.
This page page describes how to complete a production install of {{site.tseeprodname}} and connect
your {{site.tseeprodname}} cluster to an Elasticsearch cluster.

{{site.tseeprodname}} Manager users will be authenticated against {{name}} by logging in through
a [supported method]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).
{{name}} RBAC is then used to authorize queries from {{site.tseeprodname}} Manager to Elasticsearch.
From Elasticsearch's perspective all queries will come from the `tigera-ee-manager` user.

Because there's an authenticating proxy inside {{site.tseeprodname}}, any {{name}} user
given permission to access `services/calico-monitoring/elasticsearch-tigera-elasticsearch/proxy`
will be able to send queries to Elasticsearch as the `tigera-ee-manager` user.

## Before you begin

Ensure that you have followed the [installation instructions]({{site.baseurl}}/{{page.version}}/getting-started/{{path}}/installation)
up until the step to download and apply `operator.yaml`.  This document replaces
the install instructions from that point (inclusive) onwards.

To complete the following procedure, you'll need:

- An Elasticsearch cluster that meets the [requirements]({{site.baseurl}}/{{page.version}}/getting-started/{{path}}/requirements#elasticsearch-requirements).
- A `tigera-ee-fluentd` user with permission to send documents to Elasticsearch (see below).
- A `tigera-ee-manager` user with permission to issue queries to Elasticsearch (see below).
- A `tigera-ee-installer` user with permission to install machine learning jobs, and configure Kibana dashboards (see below).
- A `tigera-ee-curator` user with permission to delete indices in Elasticsearch (see below).
- The CA certificate for the Elasticsearch cluster.
- Any users who are going to use the Kibana dashboards will need to be given appropriate
  credentials.

### Setting up Elasticsearch roles

If you're using the Elasticsearch X-Pack security then you may wish to use the following roles. You should
use the [Kibana Role Management API](https://www.elastic.co/guide/en/kibana/current/role-management-api.html),
since some roles include permissions on both Kibana and Elasticsearch.

They may also be useful as a reference for defining alternative security configuration.

1. fluentd role for creating indices and sending logs to Elasticsearch

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "create_index", "write" ]
         }
       ]
     }
   }
   ```

1. {{site.tseeprodname}} Manager role for querying Elasticsearch

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "read"]
         }
       ]
     }
   }
   ```

1. {{site.tseeprodname}} role for installing machine learning jobs, Watcher jobs, and Kibana dashboards

   ```json
   {
     "elasticsearch": {
       "cluster": [ "manage_ml", "manage_watcher" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "read"]
         }
       ]
     },
     "kibana": {
       "global": ["all"]
     }
   }
   ```

1. {{site.tseeprodname}} Curator role for deleting indices older than retention period in Elasticsearch

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "all"]
         }
       ]
     }
   }
   ```
