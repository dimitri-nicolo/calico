---
title: Deploy and monitor Calico Enterprise license metrics
description: Use the Prometheus monitoring and alerting tool to get Calico Enterprise license metrics such as nodes used, nodes available, and days until license expires.
---

### Big picture

Use the Prometheus monitoring and alerting tool to get {{site.prodname}} license metrics.

### Value

Platform engineering teams need to report licensing usage on the third-party software (like {{site.prodname}}) for their CaaS/Kubernetes platforms. This is often driven by compliance, but also to mitigate risks from license expiration or usage that may impact operations. For teams to easily access these vital metrics, {{site.prodname}} provides license metrics using the Prometheus monitoring and alerting tool.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **LicenseKey API**

### Concepts

#### About Prometheus

The Prometheus monitoring tool scrapes metrics from instrumented jobs and displays time series data in a visualizer (such as Grafana). For {{site.prodname}}, the “jobs” that Prometheus can harvest metrics from the License Agent component. 


#### About License Agent

The **License Agent** is a containerized application that monitors the following {{site.prodname}} licensing information from the Kubernetes cluster, and exports the metrics through the Prometheus server:

- Days till expiration
- Nodes available
- Nodes used

#### FAQ

**Is license metrics are available through {{site.prodname}} Manager?**

 No. Currently, the only interface is prometheus. 

**How long does it take to get a new {{site.prodname}} license?**

 After receiving a Sales purchase order, approximately 1-2 days.

**What happens to nodes during the license grace period?**

 All {{site.prodname}} features will work without interruption.

**What happens to nodes after the license grace period?**
- Only policies in the default Kubernetes tier are applied
- Users can still access the {{site.prodname}} Manager, but data collection and flow logs will stop working
- {{site.prodname}} Manager displays an error message to change the license.

**What happens if I add nodes beyond what I'm licensed for?**
- New nodes that you add past your limit, are still added
- All {{site.prodname}} features still work


### How to

- [Deploy license agent in your Kubernetes cluster](#deploy-license-agent-in-your-kubernetes-cluster)
- [Create alerts using prometheus metrics](#create-alerts-using-prometheus-metrics)


#### Deploy license agent in your Kubernetes cluster

To add the license-agent component in a Kubernetes cluster for license metrics, install pull secret and apply license-agent manifest. 

1. Create namespace for license-agent

```
kubectl create namespace tigera-license-agent
```

2. Install your pull secret

```
   kubectl create secret generic tigera-pull-secret \
     --from-file=.dockerconfigjson=<path/to/pull/secret> \
     --type=kubernetes.io/dockerconfigjson -n tigera-license-agent
```

3. Apply manifest

```
kubectl apply -f {{ "/manifests/licenseagent.yaml" | absolute_url }}
```

#### Create alerts using prometheus metrics

In the following example, Alert is configured to trigger when the license expires in less than 15 days

```
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: calico-prometheus-license
  namespace: tigera-prometheus
  labels:
    role: tigera-prometheus-rules
    prometheus: calico-node-prometheus
spec:
  groups:
  - name: tigera-license.rules
    rules:
    - alert: CriticalLicenseExpiry
      expr: license_number_of_days < 15
      labels:
        severity: Warning
      annotations:
        summary: "Calico Enterprise License expires in less than 15 days"
        description: "Calico Enterprise License expires in less than 15 days"
```

### Note

Monitoring {{site.prodname}} license metrics works only with Operator based Install.
If Kubernetes api-server serves on any other port than 6443 or 443, Add that port in the Egress policy of license agent manifest.

### Above and beyond

- [LicenseKey API]({{site.baseurl}}/reference/resources/licensekey)
- [Configuring Alertmanager]({{site.baseurl}}/reference/other-install-methods/security/configuration/alertmanager)

