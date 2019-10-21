{% if include.orch != "openshift" %}
  {% assign path = "reference/other-install-methods/kubernetes" %}
  {% assign reqpath = "getting-started/kubernetes" %}
  {% assign name = "Kubernetes" %}
{% else %}
  {% assign path = "getting-started/openshift" %}
  {% assign reqpath = "getting-started/openshift" %}
  {% assign name = "OpenShift" %}
{% endif %}
{% unless include.upgrade %}
## About using your own Elasticsearch

For production clusters, you must use your own Elasticsearch cluster. We bundle an Elasticsearch
operator for quickstart and demonstration purposes, but it is not suitable for production.
This page page describes how to complete a production install of {{site.prodname}} and connect
your {{site.prodname}} cluster to an Elasticsearch cluster.

{{site.prodname}} Manager users will be authenticated against {{name}} by logging in through
a [supported method]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).
{{name}} RBAC is then used to authorize queries from {{site.prodname}} Manager to Elasticsearch.
From Elasticsearch's perspective all queries will come from the `tigera-ee-manager` user.

Because there's an authenticating proxy inside {{site.prodname}}, any {{name}} user
given permission to access `services/calico-monitoring/elasticsearch-tigera-elasticsearch/proxy`
will be able to send queries to Elasticsearch as the `tigera-ee-manager` user.

## Before you begin

Ensure that you have followed the [installation instructions](/{{page.version}}/{{path}}/installation)
up until the step to download and apply `operator.yaml`.  This document replaces
the install instructions from that point (inclusive) onwards.

To complete the following procedure, you'll need:

- An Elasticsearch cluster that meets the [requirements](/{{page.version}}/{{reqpath}}/requirements#elasticsearch-requirements).
- A `tigera-ee-fluentd` user with permission to send documents to Elasticsearch (see below).
- A `tigera-ee-manager` user with permission to issue queries to Elasticsearch (see below).
- A `tigera-ee-installer` user with permission to install machine learning jobs, and configure Kibana dashboards (see below).
- A `tigera-ee-curator` user with permission to delete indices in Elasticsearch (see below).
- A `tigera-ee-intrusion-detection` user with permission to process threat feeds, flow logs and security events (see below).
- A `tigera-ee-compliance-benchmarker` user with permission to issue queries to Elasticsearch (see below).
- A `tigera-ee-compliance-controller` user with permission to issue queries to Elasticsearch (see below).
- A `tigera-ee-compliance-reporter` user with permission to query and send documents to Elasticsearch (see below).
- A `tigera-ee-compliance-snapshotter` user with permission to query and send documents to Elasticsearch (see below).
- A `tigera-ee-compliance-server` user with permission to issue queries to Elasticsearch (see below).
- The CA certificate for the Elasticsearch cluster.
- Any users who are going to use the Kibana dashboards will need to be given appropriate
  credentials.
{% else %}

This upgrade guide assumes you are using your own Elasticsearch cluster.  (We bundle an Elasticsearch
operator for quickstart and demonstration purposes, but it is not suitable for production, and we do
not support upgrading it.)

{% endunless %}

### Setting up Elasticsearch roles

If you're using the Elasticsearch X-Pack security then you may wish to use the following roles. You should
use the [Kibana Role Management API](https://www.elastic.co/guide/en/kibana/6.4/role-management-api.html),
since some roles include permissions on both Kibana and Elasticsearch.

They may also be useful as a reference for defining alternative security configuration.

1. fluentd role for creating indices and sending logs to Elasticsearch  (`tigera-ee-fluentd`)

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

1. {{site.prodname}} Manager role for querying Elasticsearch (`tigera-ee-manager`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*", ".kibana" ],
           "privileges": [ "read"]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} role for installing machine learning jobs, Watcher jobs, and Kibana dashboards (`tigera-ee-installer`)

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
     "kibana": [
       {
         "privileges": ["all"]
       }
     ]
   }
   ```

1. {{site.prodname}} Curator role for deleting indices older than retention period in Elasticsearch (`tigera-ee-curator`)

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

1. {{site.prodname}} intrusion detection controller role for processing threat feeds, flow logs and security events. (`tigera-ee-intrusion-detection`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates"],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "read"]
         },
         {
           "names": [ ".tigera.ipset.*", "tigera_secure_ee_events.*" ],
           "privileges": [ "all"]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} compliance report and dashboard for assessing the compliance posture of the cluster.

    1. Compliance benchmarker role for storing benchmark results  (`tigera-ee-compliance-benchmarker`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_benchmark_results.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance controller role for querying last archived reports  (`tigera-ee-compliance-controller`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance reporter role for querying archived audit information and storing reports  (`tigera-ee-compliance-reporter`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_audit_*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_snapshots.*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_benchmark_results.*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance snapshotter role for recording daily configuration audits  (`tigera-ee-compliance-snapshotter`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_snapshots.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance server role for querying archived reports (`tigera-ee-compliance-server`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "read" ]
             }
           ]
         }
       }
       ```
