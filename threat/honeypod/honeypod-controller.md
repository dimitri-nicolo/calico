---
title: Monitor honeypods
description: Monitor honeypod behavior to gain insight on what attackers are doing.
canonical_url: /threat/honeypod/honeypod-controller
---

### Big picture

Monitor honeypod behavior to gain insight on what attackers are doing using packet-level inspection.

### Value

Adding monitoring to honeypods improves your ability to detect and confirm known threats by analyzing and alerting on network traffic to honeypods that match any Intrusion Detection System (Snort) signatures.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **PacketCapture** with **Honeypod controller**

### Concepts

#### About monitoring honeypods

Honeypods can optionally be monitored using a {{site.prodname}} controller that periodically polls selected honeypods for suspicious activity and scans its traffic. Alerts are generated in the Events tab of {{site.prodname}} Manager UI.

The controller leverages the following:

- [Packet capture feature]({{site.baseurl}}/visibility/packetcapture) to collect honeypod traffic in clusters.
- Open source {% include open-new-window.html text='Snort' url='https://www.snort.org/' %} to scan honeypod traffic.

### Before you begin

#### Required

[Honeypods are configured]({{site.baseurl}}/threat/honeypod/honeypods) for clusters, and alerts are generated when the honeypods are accessed.

### How To

  - [Enable packet capture on honeypods](#enable-packet-capture-on-honeypods)
  - [Add honeypod controller to cluster](#add-honeypod-controller-to-cluster)
  - [Verify honeypod controller](#verify-honeypod-controller)

#### Enable packet capture on honeypods

The following manifest enables packet capture on default [honeypods]({{site.baseurl}}/threat/honeypod/honeypods). 
Be sure to modify the namespace and selector if honeypods are placed elsewhere. 
For help, see [PacketCapture]({{site.baseurl}}/visibility/packetcapture).

```bash
kubectl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: PacketCapture
metadata:
  name: capture-honey
  namespace: tigera-internal
spec:
  selector: all()
EOF
```

In order for the honeypod controller to find the packet captures, the name `capture-honey` is required for the PacketCapture resource.

#### Add honeypod controller to cluster

> **Note**: If you’ve customized or created your own honeypods, be sure to modify the included `capture-honey` [PacketCapture]({{site.baseurl}}/visibility/packetcapture) manifest to target your honeypods. Honeypod controller requires the name to be `capture-honey` at this release.

Add the honeypod controller to each cluster configured for honeypods using the following command:

```bash
kubectl apply -f {{ "/manifests/threatdef/honeypod/controller.yaml" | absolute_url }} 
```

For Openshift deployments, the controller will require privileged access. A separate manifest is provided:

```bash
kubectl apply -f {{ "/manifests/threatdef/honeypod/controller_os.yaml" | absolute_url }} 
```

#### Adding custom signatures into Snort

By default Snort's community rule is used. Users can add their own custom signatures into the controller via ConfigMap.

The following manifest provides the method to add individual custom signatures:

```bash
kubectl create -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: localrule
  namespace: tigera-intrusion-detection
data:
  rules: |
    alert icmp any any -> any any (msg:"ICMP Echo Request"; itype:8; sid:1000000;)
    alert icmp any any -> any any (msg:"ICMP Echo Reply"; itype:0; sid:1000001;)
EOF
```

Users can also add a Snort compatible signature pack:

```bash
kubectl create cm localrule -n tigera-intrusion-detection --from-file=rules=<SIGNATURE_PACK_LOCATION>
```

> **Note**: ConfigMaps has a size limit of 1 MiB. If more space is required, use an alternate volume mount method. 

The controller deployment manifest will need to be updated to include the ConfigMap. 

Refer to the patch file below:

```bash
cat <<EOF > patch.yaml
spec:
  template:
    spec: 
      containers:
      - name: controller
        volumeMounts:
        - mountPath: /etc/snort/rules/custom.rules
          subPath: custom.rules
          name: custom-rules
          readOnly: true
      volumes:
      - name: custom-rules
        configMap:
          name: localrule
          items:
          - key: "rules"
            path: "custom.rules"
EOF
```

> **Note**: The mountPath `/etc/snort/rules/custom.rules` is required and should not be changed.

Apply the patch to the `honeypod-controller` DaemonSet:

```bash
kubectl patch daemonset honeypod-controller -n tigera-intrusion-detection --patch "$(cat patch.yaml)"
```


#### Verify honeypod controller

To verify the installation, ensure that honeypod controller is running within the `tigera-intrusion-detection` namespace:

```bash
kubectl get pods -n tigera-intrusion-detection
```

```shell
NAME                                             READY   STATUS      RESTARTS   AGE
honeypod-controller-57vwk                        1/1     Running     0          22s
honeypod-controller-8vtj6                        1/1     Running     0          22s
honeypod-controller-gk524                        1/1     Running     0          22s
honeypod-controller-k9nz4                        1/1     Running     0          22s
intrusion-detection-controller-bf9794dd7-5qxjs   1/1     Running     0          15m
intrusion-detection-es-job-installer-nfd7t       0/1     Completed   0          15m
```

### Above and beyond

- [Packet capture]({{site.baseurl}}/visibility/packetcapture)

