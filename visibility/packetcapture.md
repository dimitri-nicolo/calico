---
title: Packet capture
description: Capture live traffic for debugging microservices and application interaction.
canonical_url: /visibility/packetcapture
---
>**Note**: This feature is tech preview. Tech preview features may be subject to significant changes before they become GA.
{: .alert .alert-info}

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

### Before you begin

**FAQ**

This feature is in a technical preview stage. PacketCapture does not support:

- Enhanced filtering by selecting protocols and specific ports
- Capping a capture using either time or size
- Storing traffic in pcapng traffic
- Capturing traffic from a multi-nic setup
- Capture traffic from Calico nodes running on Windows hosts
- Capture traffic from installations using a CNI other than Calico

### How To

- [Capture live traffic](#capture-live-traffic)
- [Configure packet capture rotation](#configure-packet-capture-rotation)
- [Enforce RBAC for packet capture](#enforce-rbac-for-packet-capture)
- [Access packet capture files](#access-packet-capture-files)

#### Capture live traffic


<iframe width="260" height="127" src="https://www.youtube.com/embed/bKTkvywT7s4" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>


Capturing live traffic will start by creating a [PacketCapture]({{site.baseurl}}/reference/resources/packetcapture) resource.

Create a yaml file containing one or more packet captures and apply the packet capture to your cluster.

```bash
kubectl apply -f <your_packet_capture_filename>
```

In order to stop capturing traffic, delete the packet capture from your cluster.

```bash
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

#### Configure packet capture rotation

Live traffic will be stored as pcap files that will be rotated by size and time. All packet capture files rotate using
parameters defined in [FelixConfig]({{site.baseurl}}/reference/resources/felixconfig).

Packet Captures files will be rotated either when reaching maximum size or when passing rotation time.

For example, in order to extend the time rotation to one day, the command below can be used:

```bash
kubectl patch felixconfiguration default -p '{"spec":{"captureRotationSeconds":"86400"}}'
```

#### Enforce RBAC for packet capture

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

In order to allow user jane to access the capture files generated for a specific namespace, a role/role binding similar to the one below can be used:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tigera-authentication-clusterrole-jane
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["authenticationreviews"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tigera-authentication-clusterrolebinding-jane
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tigera-authentication-clusterrole-jane
subjects:
- kind: ServiceAccount
  name: jane
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: sample
  name: tigera-capture-files-role
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["packetcaptures/files"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tigera-capture-files-role-jane
  namespace: sample
subjects:
- kind: ServiceAccount
  name: jane
  namespace: default
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: tigera-capture-files-role
  apiGroup: rbac.authorization.k8s.io
```

#### Access packet capture files

Capture files will be stored on the host mounted volume used for calico nodes. These can be visualized using tools such as Wireshark.

In order to locate the capture files generated, query the status of the [PacketCapture]({{site.baseurl}}/reference/resources/packetcapture)

```bash
export NS=<REPLACE_WITH_CAPTURE_NAMESPACE>
export NAME=<REPLACE_WITH_CAPTURE_NAME>
```

```bash
kubectl get packetcaptures -n $NS $NAME -o yaml
```

Sample of received output:
```
apiVersion: projectcalico.org/v3
kind: PacketCapture
metadata:
  name: sample-capture-all
  namespace: sample
spec:
  selector: all()
status:
  files:
  - directory: /var/log/calico/pcap
    fileNames:
    - pod_cali.pcap
    node: node-0
```

To access the capture files locally, you can use the following api that is available via tigera-manager service:

```bash
kubectl port-forward -n tigera-manager service/tigera-manager 9443:9443 &
NS=<REPLACE_WITH_PACKETCAPTURE_NS> NAME=<REPLACE_WITH_PACKETCAPTURE_NAME> TOKEN=<REPLACE_WITH_YOUR_TOKEN> \
curl "https://localhost:9443/packet-capture/download/$NS/$NAME/files.zip" \
-H "Authorizaton: Bearer $TOKEN"
```

Retrieving capture files from a managed cluster is performed by calling the same API:

```bash
kubectl port-forward -n tigera-manager service/tigera-manager 9443:9443 &
NS=<REPLACE_WITH_PACKETCAPTURE_NS> NAME=<REPLACE_WITH_PACKETCAPTURE_NAME> TOKEN=<REPLACE_WITH_YOUR_TOKEN> MANAGED_CLUSTER=<REPLACE_WITH_THE_NAME_OF_MANAGED_CLUSTER>\
curl "https://localhost:9443/packet-capture/download/$NS/$NAME/files.zip" \
-H "Authorizaton: Bearer $TOKEN" -H "X-CLUSTER-ID: $MANAGED_CLUSTER"
```

Next, get the token from the service account.
Using the running example of a service account named, `jane` in the default namespace:

```bash
{% raw %}kubectl get secret $(kubectl get serviceaccount jane -o jsonpath='{range .secrets[*]}{.name}{"\n"}{end}' | grep token) -o go-template='{{.data.token | base64decode}}' && echo{% endraw %}
```

Alternatively, you can access the capture files locally using [calicoctl]({{site.baseurl}}/reference/calicoctl/captured-packets) CLI:

```bash
calicoctl captured-packets copy sample-capture -namespace sample --destination /tmp
```

You can access the capture files locally from the Fluentd pods using similar commands like the ones below:

```bash
kubectl get pods -ntigera-fluentd --no-headers --field-selector spec.nodeName="<REPLACE_WITH_NODE_NAME>"
```

```bash
kubectl cp tigera-fluentd/<REPLACE_WITH_POD_NAME>:var/log/calico/pcap/sample/sample-capture/ .
```

Packet capture files will be stored using the following directory structure: {namespace}/{packet capture resource name} under the capture directory defined via FelixConfig.
The active packet capture file will be identified using the following schema: {workload endpoint name}_{host network interface}.pcap. Rotated capture files name will contain an index matching the rotation timestamp.

Packet capture files will not be deleted after a capture has stopped. 

[calicoctl]({{site.baseurl}}/reference/calicoctl/captured-packets) CLI can be used to clean capture files:

```bash
calicoctl captured-packets clean sample-capture -namespace sample
```

Alternatively, the following command can be used to clean up capture files:

```bash
kubectl exec -it tigera-fluentd/<REPLACE_WITH_POD_NAME> -- sh -c "rm -r /var/log/calico/pcap/sample/sample-capture/"
```
