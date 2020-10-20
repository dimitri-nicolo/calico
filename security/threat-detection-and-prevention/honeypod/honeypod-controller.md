---
title: Using Honeypod controller to provide packet inspection on suspisious traffic
description: Monitor Honeypods and scan traffic with a packet inspection engine
canonical_url: /security/threat-detection-and-preventaion/honeypod-controller
---

### Big picture

[Honeypods]({{site.baseurl}}/security/threat-detection-and-preventaion/honeypods) provides a way to detect compromised resources within your cluster, Honeypod controller can be deployed on top of Honeypods to monitor its behavior and gain insight on what the attackers is doing.

### Value

The controller extends Honeypods by providing a way to monitor suspicious traffic reaching the Honeypods and generate alerts if the traffic matches any Intrustion Detection System(Snort) signatures. If the controller detects activities within the Honeypods, it will trigger an Snort scan on the traffic reaching the Honeypods. 


### Features

This how-to guide uses the following Calico Enterprise features:

- **PacketCapture** with **Honeypod controller**

### Concepts

The Honeypod controller leverage the [PacketCapture]({{site.baseurl}}/security/threat-detection-and-preventaion/packetcapture) feature to collect traffic related to Honeypods deployed in the cluster. Snort is used to scan the packet captures and alerts created can be found in the Events page of Calico Enterprise.

### How To

- [Big picture](#big-picture)
- [Value](#value)
- [Features](#features)
- [Concepts](#concepts)
- [How To](#how-to)
  - [Ensure Honeypods are deployed](#ensure-honeypods-are-deployed)
  - [Deploy RBAC used by Honeypod controller](#deploy-rbac-used-by-honeypod-controller)
  - [Enable packet capture on Honeypods](#enable-packet-capture-on-honeypods)
  - [Deploy Honeypod controller manifest](#deploy-honeypod-controller-manifest)
- [Usage](#usage)

#### Ensure Honeypods are deployed

Review [Honeypods]({{site.baseurl}}/security/threat-detection-and-preventaion/honeypods) installation guide and that Alerts are generated when the Honeypods are accessed.

#### Deploy RBAC used by Honeypod controller

Create Service Account and ClusterRoleBinding for  Honeypod Controller.

    ```shell
    apiVersion: v1
    kind: ServiceAccount
    metadata:
    name: honeypod-controller 
    namespace: tigera-intrusion-detection
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
    namespace: tigera-intrusion-detection
    name: honeypod-controller-role
    rules:
    - apiGroups: ["projectcalico.org"]
        resources: ["workloadendpoints"]
        verbs: ["get"]
    - apiGroups: [""]
        resources: ["pods"]
        verbs: ["list","get"]
    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
    name: honeypod-controller-binding
    subjects:
    - kind: ServiceAccount
        name: honeypod-controller 
        namespace: tigera-intrusion-detection
    roleRef:
    kind: ClusterRole
    name: honeypod-controller-role
    apiGroup: rbac.authorization.k8s.io
    ```

#### Enable packet capture on Honeypods

Create a PacketCapture resource and use selector to select the Honeypods. Here we select the `tigera-internal` namespace:

    ```shell
    apiVersion: projectcalico.org/v3
    kind: PacketCapture
    metadata:
    name: capture-honey
    namespace: tigera-internal
    spec:
    selector: all()
    ```

#### Deploy Honeypod controller manifest

Deploy a DaemonSet of Honeypod Controller pods into the cluster.

    ```shell
    apiVersion: apps/v1
    kind: DaemonSet
    metadata:
    name: honeypod-controller
    namespace: tigera-intrusion-detection
    spec:
    selector:
        matchLabels:
        app: honeypod-controller
    template:
        metadata:
        labels:
            app: honeypod-controller
        spec:
        serviceAccountName: honeypod-controller
        imagePullSecrets:
        - name: tigera-pull-secret
        containers:
        - name: controller 
            imagePullPolicy: "IfNotPresent"
            image: quay.io/tigera/honeypod-controller:master
            env:
            - name: ELASTIC_SCHEME
            value: https
            - name: ELASTIC_HOST
            value: tigera-secure-es-http.tigera-elasticsearch.svc
            - name: ELASTIC_PORT
            value: "9200"
            - name: ELASTIC_SSL_VERIFY
            value: "true"
            - name: ELASTIC_USER
            valueFrom:
                secretKeyRef:
                key: username
                name: tigera-ee-intrusion-detection-elasticsearch-access
            - name: ELASTIC_USERNAME
            valueFrom:
                secretKeyRef:
                key: username
                name: tigera-ee-intrusion-detection-elasticsearch-access
            - name: ELASTIC_PASSWORD
            valueFrom:
                secretKeyRef:
                key: password
                name: tigera-ee-intrusion-detection-elasticsearch-access
            - name: ELASTIC_CA
            value: /etc/ssl/elastic/ca.pem
            - name: ES_CA_CERT
            value: /etc/ssl/elastic/ca.pem
            - name: NODENAME
            valueFrom:
                fieldRef:
                fieldPath: spec.nodeName
            volumeMounts:
            - mountPath: /etc/ssl/elastic/
            name: elastic-ca-cert-volume
            readOnly: true
            - mountPath: /pcap
            name: pcap
            readOnly: true
            - mountPath: /snort
            name: snort
        volumes:
        - name: elastic-ca-cert-volume
            secret:
            defaultMode: 420
            items:
            - key: tls.crt
                path: ca.pem
            secretName: tigera-secure-es-http-certs-public
        - name: pcap
            hostPath:
            path: /var/log/calico/pcap/
            type: Directory 
        - name: snort
            emptyDir: {}
    ```

### Usage

The Honeypod Controller will periodically poll for any suspisous activity in the selected Honeypods and scans its traffic. Any alerts generated will be in the Events tab of Calico Enterprise.