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

Ensure that you have followed the [installation instructions]({{site.baseurl}}/{{page.version}}/getting-started/{{path}}/installation)
up until the step to download and apply `operator.yaml`.  This document replaces
the install instructions from that point (inclusive) onwards.

To complete the following procedure, you'll need:

- An Elasticsearch cluster that meets the [requirements]({{site.baseurl}}/{{page.version}}/getting-started/{{path}}/requirements#elasticsearch-requirements).
- A `tigera-ee-fluentd` user with permission to send documents to Elasticsearch.
- A `tigera-ee-manager` user with permission to issue queries to Elasticsearch.
- The CA certificate for the Elasticsearch cluster.
- Any users who are going to use the Kibana dashboards will need to be given appropriate
  credentials.