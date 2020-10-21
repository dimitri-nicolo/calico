---
title: Honeypods as canary threat detection
description: Use of canary resources to detect compromised workloads
canonical_url: /security/threat-detection-and-prevention/honeypod/honeypods
---

### Big picture

Use Honeypods to get alerts that indicate a compromised resource is present within your cluster.

### Value

Honeypods are customizable canary pods or resources placed in a cluster such that all defined and valid resources should never attempt to access or make connections to them. If any resources reach these Honeypods, we can automatically assume the connection is suspicious at minimum and that the source resource may have been compromised.

### Features

This how-to guide uses the following Calico Enterprise features:

- **GlobalAlerts** with **Honeypods**

### Concepts

- Honeypods are deployed on a per-cluster basis, via a set of manifests
- The naming for all Honeypods resources are customizable
- Any communication attempts to these pods will trigger an alert and be found in the Alerts tab of the Calico Enterprise Manager UI
- A Kibana Dashboard is also provided for an overview of the triggered alerts

### How To

  - [Setting up resources](#setting-up-resources)
  - [Deploying Honeypods](#deploying-honeypods)

#### Setting up resources

Apply the manifest to create a namespace and RBAC for the Honeypods: 

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/common.yaml" | absolute_url }} 
```

Add `tigera-pull-secret` into the namespace `tigera-internal`:

```shell
kubectl create secret generic tigera-pull-secret --from-file=.dockerconfigjson=<pull-secrets.json> --type=kubernetes.io/dockerconfigjson -n tigera-internal
```

#### Deploying Honeypods

A set of example Honeypods and their use cases have been created by Tigera. All images contain a minimal container which runs or mimics a running application.

1. IP Enumeration. By not setting a service, the pod can only be reach locally (adjacent pods within same node):

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/ip-enum.yaml" | absolute_url }} 
```

2. Exposed Service (nginx). Expose a nginx service that serves a generic page. The pod can be discovered via ClusterIP or DNS lookup. An unreachable service `tigera-dashboard-internal-service` is created to entice the attacker in finding and reaching to `tigera-dashboard-internal-debug`:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/expose-svc.yaml" | absolute_url }} 
```

3. Vulnerable Service (MySQL). Expose a SQL service that contains an empty database with easy (root, no password) access. The pod can be discovered via ClusterIP or DNS lookup:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/vuln-svc.yaml" | absolute_url }} 
```

### Customizations 

Honeypods are flexible and can be customized easily. The namespace `tigera-internal`, as well as the pods and service names can be changed to disguise the Honeypods further. The manifests will only need to be updated to ensure the GlobalAlert resources can still monitor the network flows.

### Visualization 

Any alerts triggered will be shown in the Alerts tab in Calico Enterprise. A dashboard in Kibana named `Honeypod Dashboard` provides an easy way to monitor and analyze traffic reaching the Honeypods.