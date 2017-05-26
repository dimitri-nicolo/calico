---
title: Configuring Prometheus
---

#### Storage

The sample manifests does not define a _Storage_ spec. This means that if you
use the default manifest, Prometheus will be deploymed with a `emptyDir`
volume. Using a (persistent volume)[https://kubernetes.io/docs/concepts/storage/persistent-volumes/]
supported by Kubernetes is strongly recommended.

The _Prometheus_ third party resource defined by the _Prometheus Operator_
supports defining a _StorageClass_

As an example, to run a _Prometheus_ instance in GCE, first define a
_StorageClass_ spec, like so:

	kind: StorageClass
	apiVersion: storage.k8s.io/v1
	metadata:
	  name: ssd
	provisioner: kubernetes.io/gce-pd
	parameters:
	  type: pd-ssd
	  zone: us-west1-b

Then add the `storage` field to the _Prometheus_ manifest.

	apiVersion: monitoring.coreos.com/v1alpha1
	kind: Prometheus
	metadata:
	  name: calico-node-prometheus
	  namespace: calico-monitoring
	spec:
	  serviceAccountName: prometheus
	  serviceMonitorSelector:
	    matchLabels:
	      team: network-operators
	  version: v1.6.3
	  retention: 1h
	  resources:
	    requests:
	      memory: 800Mi
	  ruleSelector:
	    matchLabels:
	      role: calico-prometheus-rules
	      prometheus: calico-node-prometheus
	  alerting:
	    alertmanagers:
	      - namespace: calico-monitoring
	        name: calico-node-alertmanager
	        port: web
	        scheme: http
	  storage:
	    class: ssd
	    resources:
	      requests:
	        storage: 80Gi


_Prometheus Operator_ also supports other manual storage provisioning
mechanisms. More information can be found [here](https://github.com/coreos/prometheus-operator/blob/d360a3bae5ab054732837d5a715814cd1c78d745/Documentation/user-guides/storage.md).

Combining storage resource with proper retention time for metrics will ensure
that Prometheus will use the storage effectively. The `retention` field is used
to configure the amount of time that metrics are stored on disk.

### Crashes, Restarts and Metrics Availability

Prometheus stores metrics at regular intervals. If Prometheus is restarted and
if ephemeral storage is used, metrics will be lost. Use Kubernetes Persistent
Volumes to ensure that stored data will be accessible across restart/crashes.

