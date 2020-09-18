---
title: Capture packets for debugging workloads
description: Capture live traffic for debugging microservices and application interaction.
canonical_url: /security/threat-detection-and-prevention/packetcapture
---
>**Warning**: This feature is experimental.  Experimental features may change significantly or not be supported in future releases.
{: .alert .alert-danger}

### Big picture

Capture live traffic inside a Kubernetes cluster, and export to visualization tools like Wireshark for troubleshooting and debugging applications.

### Value 

Packet capture is a valuable tool for debugging microservices and application interaction in day-to-day operations and incident response. But manually setting up packet capturing can be tedious. Calico Enterprise provides an easy way to capture packets using the widely-known "pcap" format, and export them to visualization tools like WireShark.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **FelixConfig** with **PacketCapture**

### Concepts

Libpcap file format, also known as [pcap](https://wiki.wireshark.org/Development/LibpcapFileFormat), is the main file 
format used for capturing traffic by network tools.

### Before you begin...

**FAQ**

This feature is in a technical preview stage. PacketCapture does not support:

- Enhanced filtering by selecting protocols and specific ports
- Capping a capture using either time or size
- Storing traffic in pcapng traffic
- Capturing traffic from a multi-nic setup

### How To

- [Capturing live traffic](#capturing-live-traffic)
- [Configure Packet Capture rotation](#configure-packet-capture-rotation)
- [Enforce RBAC for Packet Capture](#enforce-rbac-for-packet-capture)
- [Access packet capture files](#access-packet-capture-files)
- [Troubleshooting](#troubleshooting)

#### Capturing live traffic

Capturing live traffic will start by creating a [PacketCapture]({{site.baseurl}}/reference/resources/packetcapture) resource.

Create a yaml file containing one or more packet captures and apply the packet capture to your cluster.

```shell
kubectl apply -f <your_packet_capture_filename>
```

In order to stop capturing traffic, delete the packet capture from your cluster.

```shell
kubectl delete -f <your_packet_capture_filename>
```
**Examples of selecting workloads**

Following is a basic example to select a single workload that has the label `k8s-app` with value `nginx`.

```yaml
apiVersion: projectcalico.org/v3
kind: PacketCapture
metadata:
  name: sample-capture-nginx
  namespace: sample
spec:
  selector: k8s-app == "nginx"
```

In the following example, we select all workload endpoints in `sample` namespace.

```yaml
apiVersion: projectcalico.org/v3
kind: PacketCapture
metadata:
  name: sample-capture-all
  namespace: sample
spec:
  selector: all()
```

#### Configure Packet Capture rotation

Live traffic will be stored as pcap files that will be rotated by size and time. All packet capture files rotate using
parameters defined in [FelixConfig]({{site.baseurl}}/reference/resources/felixconfig).

Packet Captures files will be rotated either when reaching maximum size or when passing rotation time.

For example, in order to extend the time rotation to one day, the command below can be used:

```shell
kubectl patch felixconfiguration default -p '{"spec":{"captureRotationSeconds":"86400"}}'
```

#### Enforce RBAC for Packet Capture

Packet Capture permissions are enforced using the standard Kubernetes RBAC based on Role and RoleBindings within a namespace.

For example, in order to allow user jane to create/delete/get/list/update/watch packet captures for a specific namespace, the command below can be used:
 
```
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: sample
  name: tigera-packet-capture-role
rules:
- apiGroups: ["projectcalico.org"] 
  resources: ["packetcaptures"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tigera-packet-capture-role-jane
  namespace: sample
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: tigera-packet-capture-role
  apiGroup: rbac.authorization.k8s.io
```

#### Access packet capture files

Capture files will be stored on the host mounted volume used for calico nodes. These can be visualized using tools such as Wireshark.

To access the capture files locally, you must extract the files from the Fluentd pods similar to the below commands:

```shell
kubectl get pods -A -l <REPLACE_WITH_LABEL_SELECTOR> -o jsonpath="{..nodeName}"
```

```shell
kubectl get pods -ntigera-fluentd --no-headers --field-selector spec.nodeName="<REPLACE_WITH_NODE_NAME>"
```

```shell
kubectl cp tigera-fluentd/<REPLACE_WITH_POD_NAME>:var/log/calico/pcap/sample/sample-capture/ .
```

Packet capture files will be stored using the following directory structure: {namespace}/{packet capture resource name} under the capture directory defined via FelixConfig.
The active packet capture file will be identified using the following schema {workload endpoint name}_{host network interface}.pcap. Rotated capture files name will contain an index matching the rotation timestamp.

Packet capture files will not be deleted after a capture has stopped. The following command can be used to clean up capture files:

```shell
kubectl exec -it tigera-fluentd/<REPLACE_WITH_POD_NAME -- sh -c "rm -r /var/log/calico/pcap/sample/sample-capture/"
```

#### Troubleshooting

To troubleshoot packet capture, see [Troubleshooting]({{site.baseurl}}/maintenance/troubleshoot/troubleshooting)
