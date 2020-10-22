---
title: Configure honeypods
description: Configure honeypods to detect compromised workloads.
canonical_url: /security/threat-detection-and-prevention/honeypod/honeypods
---

### Big picture

Configure honeypods in your clusters and get alerts that may indicate a compromised resource.

### Value

Based on the well-known cybersecurity method, “honeypots,” {{site.prodname}} honeypods are used to detect and counter cyber attacks. You place decoys disguised as a sensitive asset (called canary pods) at different locations in your Kubernetes cluster. You configure all valid resources to not make connections to the honeypots. Then, if any resources do reach the honeypots, you can assume the connection is suspicious, and that a resource may be compromised.

{{site.prodname}} honeypods can be used to detect attacks such as:

- Data exfiltration
- Resources enumeration
- Privilege escalation
- Denial of service
- Vulnerability exploitation attempts

### Features

This how-to guide uses the following {{site.prodname}} features:

- **GlobalAlerts** with **Honeypods**

### Concepts

#### Honeypod implementation

You configure honeypods on a per-cluster basis using "template" honeypod manifests that are easily customizable. Any alerts triggered are displayed in the Alerts tab in {{site.prodname}} Manager UI. The Honeypod Dashboard in Kibana provides an easy way to monitor and analyze traffic reaching the honeypods.

### How To

  - [Configure namespace and RBAC for honeypods](#configure-namespace-and-rbac-for-honeypods)
  - [Deploy honeypods in clusters](#deploy-honeypods-in-clusters)
  - [Troubleshooting](#troubleshooting)

#### Configure namespace and RBAC for honeypods

Apply the following manifest to create a namespace and RBAC for the honeypods: 

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/common.yaml" | absolute_url }} 
```

Add `tigera-pull-secret` into the namespace `tigera-internal`:

```shell
kubectl create secret generic tigera-pull-secret --from-file=.dockerconfigjson=<pull-secrets.json> --type=kubernetes.io/dockerconfigjson -n tigera-internal
```

#### Deploy honeypods in clusters

Use one of the following sample honeypods manifests to customize for your implementation. All images contain a minimal container that runs or mimics a running application. Note that the namespace `tigera-internal`, as well as the pods and service names can be changed to disguise the honeypods further.

> Note: If you change the manifest or customize a honeypod, you must update the [globalalert manifest]({{site.baseurl}}/reference/resources/globalalert).

- **IP Enumeration** 
The pod can only be reached locally (adjacent pods within same node), by not setting a service:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/ip-enum.yaml" | absolute_url }} 
```

- **Exposed service (nginx)**
Expose a nginx service that serves a generic page. The pod can be discovered via ClusterIP or DNS lookup. An unreachable service `tigera-dashboard-internal-service` is created to entice the attacker to find and reach, `tigera-dashboard-internal-debug`:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/expose-svc.yaml" | absolute_url }} 
```

- **Vulnerable Service (MySQL)**
Expose a SQL service that contains an empty database with easy access. The pod can be discovered via ClusterIP or DNS lookup:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/vuln-svc.yaml" | absolute_url }} 
```

#### Troubleshooting

To troubleshoot honeypods, see [Troubleshooting]({{site.baseurl}}/maintenance/troubleshoot/troubleshooting)