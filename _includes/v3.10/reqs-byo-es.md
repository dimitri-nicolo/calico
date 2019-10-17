## Elasticsearch requirements

For quickstart and demonstration purposes, {{site.prodname}} bundles an
[Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator) that deploys an Elasticsearch cluster
and a Kibana instance that meets the following requirements.

The operator deployment is not suitable for production. Ensure that you have an Elasticsearch
cluster and Kibana instance that complies with the following.

- Elasticsearch version {{site.data.versions[page.version].first.components["elasticsearch"].version}}.
- Kibana version {{site.data.versions[page.version].first.components["kibana"].version}}.
- Kubernetes pods must be able to reach the Elasticsearch cluster.
- The Elasticsearch cluster must support username and password authentication.
