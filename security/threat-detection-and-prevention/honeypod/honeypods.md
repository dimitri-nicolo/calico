---
title: Honeypods as canary threat detection
description: Use of canary resources to detect compromised workloads
canonical_url: /security/threat-detection-and-preventaion/honeypods
---

### Big picture

Use Honeypod to get alerts that indicate a compromised resource is present within your cluster.

### Value

Honeypod is a customisable set of canary pods or resources placed in a cluster such that all defined and valid resources should never attempt to access or make connections to them. If any resources reach these Honeypods, we can automatically assume the connection is suspicious at minimum and that the source resource may have been compromised.

### Features

This how-to guide uses the following Calico Enterprise features:

- **GlobalAlerts** with **Honeypod**

### Concepts

- Honeypod is deployed on a per-cluster basis, via a set of manifests
- The naming for all Honeypod resources is customisable
- Any communication attempts to these pods will trigger an alert via the Alerts tab in Manager UI
- A Kibana Dashboard is also provided for an overview of the triggered alerts

### How To

- [Big picture](#big-picture)
- [Value](#value)
- [Features](#features)
- [Concepts](#concepts)
- [How To](#how-to)
  - [Setting up resources](#setting-up-resources)
  - [Deploying Honeypods](#deploying-honeypods)
  - [Creating Global Alerts for detection](#creating-global-alerts-for-detection)
- [Customizations](#customizations)
- [Visualization](#visualization)

#### Setting up resources

Create a namespace and RBAC for the Honeypods. 

    ```shell
    apiVersion: v1
    kind: Namespace
    metadata:
    name: tigera-internal
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
    namespace: tigera-internal
    name: tigera-internal-role
    rules:
    - apiGroups: [""]
        resources: [""]
        verbs: [""]
    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
    name: tigera-internal-binding
    subjects:
    - kind: Group
        name: system:authenticated
        apiGroup: rbac.authorization.k8s.io
    roleRef:
    kind: ClusterRole
    name: tigera-internal-role
    apiGroup: rbac.authorization.k8s.io
    ```

#### Deploying Honeypods

A set of example Honeypods and their use cases have been created by Tigera. All images contain a mininal container which runs or minics a running application.

1. IP Enumeration. By not setting a service, the pod can only be reach locally (adjacent pods within same node).

    ```shell
    apiVersion: apps/v1
    kind: DaemonSet
    metadata:
    name: tigera-internal-1 
    namespace: tigera-internal
    spec:
    selector:
        matchLabels:
        app: tigera-internal-1
    template:
        metadata:
        labels:
            app: tigera-internal-1
        spec:
        imagePullSecrets:
        - name: tigera-pull-secret
        containers:
        - name: tigera-internal-1
            image: quay.io/tigera/honeypod:v2.7.0
    ```
2. Exposed Service (nginx). Expose a nginx service that serves a generic page. The pod can be discovered via ClusterIP or DNS lookup. An unreachable service `tigera-dashboard-internal-service` is created to entice the attacker in finding and reaching to `tigera-dashboard-internal-debug`.

    ```shell
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: tigera-internal-3
    namespace: tigera-internal
    spec:
    selector:
        matchLabels:
        app: tigera-internal-3
    replicas: 1
    template:
        metadata:
        labels:
            app: tigera-internal-3
        spec:
        imagePullSecrets:
        - name: tigera-pull-secret
        containers:
        - name: tigera-internal-dashboard
            image: quay.io/tigera/honeypod-exp-service:v2.7.0
            ports:
            - containerPort: 8888
            - containerPort: 443
    ---
    kind: Service
    apiVersion: v1
    metadata:
    name: tigera-dashboard-internal-debug
    namespace: tigera-internal 
    labels:
        app: tigera-dashboard-internal-debug
    spec:
    ports:
        - port: 8888
        protocol: TCP
        targetPort: 8080
    selector:
        app: tigera-internal-3
    ---
    kind: Service
    apiVersion: v1
    metadata:
    name: tigera-dashboard-internal-service
    namespace: tigera-internal 
    labels:
        app: tigera-dashboard-internal-service
    spec:
    ports:
        - port: 443 
        protocol: TCP
    selector:
        app: tigera-internal-3
    ```
3. Vulnerable Service (MySQL). Expose a SQL service that contains an empty database with easy (root, no password) access. The pod can be discovered via ClusterIP or DNS lookup.

    ```shell
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: tigera-internal-4
    namespace: tigera-internal
    spec:
    selector:
        matchLabels:
        app: tigera-internal-4
    replicas: 1
    template:
        metadata:
        labels:
            app: tigera-internal-4
        spec:
        imagePullSecrets:
        - name: tigera-pull-secret
        securityContext:
            runAsUser: 999
            runAsGroup: 999
        containers:
        - name: tigera-internal-4
            image: mysql:latest
            ports:
            - containerPort: 3306
            env:
            - name: MYSQL_ALLOW_EMPTY_PASSWORD 
            value: 'yes'
    ---
    kind: Service
    apiVersion: v1
    metadata:
    name: tigera-internal-backend
    namespace: tigera-internal
    labels:
        app: tigera-internal-backend
    spec:
    ports:
        - port: 3306
        protocol: TCP
        targetPort: 3306
    selector:
        app: tigera-internal-4
    ```

#### Creating Global Alerts for detection

After deploy the Honeypods, Global Alerts are created to monitor the flows reaching the Honeypods and trigger an Alert.

1. SSH access

    ```shell
    apiVersion: projectcalico.org/v3
    kind: GlobalAlertTemplate
    metadata:
    name: honeypod.network.ssh
    spec:
    description: "Alerts on Honeypod accessed via SSH"
    summary: "[Honeypod] Attempted SSH connection by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_port"="22" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ---
    apiVersion: projectcalico.org/v3
    kind: GlobalAlert
    metadata:
    name: honeypod.network.ssh
    spec:
    description: "[Honeypod] Attempted SSH connection by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_port"="22" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ```

2. IP enumeration

    ```shell
    apiVersion: projectcalico.org/v3
    kind: GlobalAlertTemplate
    metadata:
    name: honeypod.ip.enum
    spec:
    description: "Alerts on Honeypod accessed via IP enumeration"
    summary: "[Honeypod] Pod subnet IP enumeration by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-1" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ---
    apiVersion: projectcalico.org/v3
    kind: GlobalAlert
    metadata:
    name: honeypod.ip.enum
    spec:
    description: "[Honeypod] Pod subnet IP enumeration by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-1" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ```

3. Port scan

    ```shell
    apiVersion: projectcalico.org/v3
    kind: GlobalAlertTemplate
    metadata:
    name: honeypod.port.scan
    spec:
    description: "Alerts on Honeypod accessed via port scan"
    summary: "[Honeypod] Pod subnet port scan by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-1" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    metric: count
    condition: gte
    threshold: 4
    ---
    apiVersion: projectcalico.org/v3
    kind: GlobalAlert
    metadata:
    name: honeypod.port.scan
    spec:
    description: "[Honeypod] Pod subnet port scan by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-1" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    metric: count
    condition: gte
    threshold: 4
    ```

3.  Fake debug service access

    ```shell
    apiVersion: projectcalico.org/v3
    kind: GlobalAlertTemplate
    metadata:
    name: honeypod.fake.svc
    spec:
    description: "Alerts on Honeypod accessed via fake debug service"
    summary: "[Honeypod] Fake debug service accessed by ${source_namespace}/${source_name_aggr} on port 8080"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-3" AND dest_port="8080" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ---
    apiVersion: projectcalico.org/v3
    kind: GlobalAlert
    metadata:
    name: honeypod.fake.svc
    spec:
    description: "[Honeypod] Fake debug service accessed by ${source_namespace}/${source_name_aggr} on port 8080"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-3" AND dest_port="8080" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ```

4.  Fake vulnerable service access

    ```shell
    apiVersion: projectcalico.org/v3
    kind: GlobalAlertTemplate
    metadata:
    name: honeypod.vuln.svc
    spec:
    description: "Alerts on Honeypod accessed via vulnerable (MySQL) service"
    summary: "[Honeypod] Vulnerable service (MySQL) accessed by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-4" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ---
    apiVersion: projectcalico.org/v3
    kind: GlobalAlert
    metadata:
    name: honeypod.vuln.svc
    spec:
    description: "[Honeypod] Vulnerable service (MySQL) accessed by ${source_namespace}/${source_name_aggr}"
    severity: 100
    dataSet: flows
    query: 'dest_namespace="tigera-internal" AND "dest_labels.labels"="app=tigera-internal-4" AND reporter="dst"'
    aggregateBy: [source_namespace, source_name_aggr, dest_namespace, dest_name_aggr, host.keyword]
    ```

### Customizations 
Honeypods are flexible and can be customized easily. The namespace `tigera-internal`, as well as the pods and service names can be changed to disguise the Honeypods further. The manifests will only need to be updated to ensure the GlobalAlert resources can still monitor the network flows.

### Visualization 

Any alerts triggered will be shown in the Alerts tab in Calico Enterprise. A dashboard in Kibana named `Honeypod Dashboard` provides an easy way to monitor and analyze traffic reaching the Honeypods.