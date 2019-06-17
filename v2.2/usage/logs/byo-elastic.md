---
title: Using your own Elasticsearch cluster
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/byo-elasticsearch
---

This page describes how to use a separate Elasticsearch cluster for {{site.prodname}} logs.
It may be useful if you have an existing cluster you want to use, or if you want to provision
it yourself instead of using the bundled [Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator).

### Requirements

- Elasticsearch version {{site.data.versions[page.version].first.components["elasticsearch"].version}}
- Kibana version {{site.data.versions[page.version].first.components["kibana"].version}}

### Remove the Elasticsearch operator

1. Delete the Elasticsearch operator, or edit the `operator.yaml` manifest to remove it.

   ```
   kubectl delete deployment -n calico-monitoring tigera-elasticsearch
   ```

1. Delete the Elasticsearch cluster by removing it from the `monitor-calico.yaml` manifest.
   (Kubernetes will fail to create it if you removed the operator.)

1. Ensure that any network policy you've created will allow traffic from fluentd (selector
   `k8s-app: tigera-fluentd-node` in `kube-system`), and from the Kubernetes API Server
   (the web interface uses the API Server proxy to reach Elasticsearch).

### Create a service to point to your cluster

{{site.prodname}} components access Elasticsearch via a service.

Create the following service, and add your Elasticsearch cluster (or its client nodes
or load balancers as appropriate for your architecture) to that service.

```
apiVersion: v1
kind: Service
metadata:
  name: elasticsearch-tigera-elasticsearch
  namespace: calico-monitoring
spec:
  type: ClusterIP
  ports:
  - port: 9200
    targetPort: 9200
```

### Configure authentication to your cluster
{% include {{page.version}}/elastic-secure.md %}
