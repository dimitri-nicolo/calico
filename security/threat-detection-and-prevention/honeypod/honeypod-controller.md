---
title: Monitor honeypods
description: Monitor honeypod behavior to gain insight on what attackers are doing.
canonical_url: /security/threat-detection-and-prevention/honeypod/honeypod-controller
---

### Big picture

Monitor honeypod behavior to gain insight on what attackers are doing using packet-level inspection.

### Value

Adding monitoring to honeypods extends your ability to thwart activities including generating alerts if traffic to honeypods match any Intrusion Detection System (Snort) signatures, and scanning traffic if activities are detected.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **PacketCapture** with **Honeypod controller**

### Concepts

#### About monitoring honeypods

Honeypods can optionally be monitored using a {{site.prodname}} controller that periodically polls selected honeypods for suspicious activity and scans its traffic. Alerts are generated in the Events tab of {{site.prodname}} Manager UI.

The controller leverages the following:

- [Packet capture feature]({{site.baseurl}}/security/threat-detection-and-prevention/packetcapture) to collect honeypod traffic in clusters.
- Open source {% include open-new-window.html text='Snort' url='https://www.snort.org/' %} to scan honeypod traffic.

### Before you begin

#### Required

[Honeypods are configured]({{site.baseurl}}/security/threat-detection-and-prevention/honeypod/honeypods) for clusters, and alerts are generated when the honeypods are accessed.

### How To

  - [Add honeypod controller to cluster](#add-honeypod-controller-to-cluster)
  - [Verify honeypod controller deployment](#verify-honeypod-controller-deployment)

#### Add honeypod controller to cluster

Add the honeypod controller to each cluster configured for honeypods using the following command:

```bash
kubectl apply -f {{ "/manifests/threatdef/honeypod/controller.yaml" | absolute_url }} 
```

#### Verify honeypod controller deployment

To verify the installation, ensure that honeypod controller is running within the `tigera-intrusion-detection` namespace:

```shell
$ kubectl get pods -n tigera-intrusion-detection
NAME                                             READY   STATUS      RESTARTS   AGE
honeypod-controller-57vwk                        1/1     Running     0          22s
honeypod-controller-8vtj6                        1/1     Running     0          22s
honeypod-controller-gk524                        1/1     Running     0          22s
honeypod-controller-k9nz4                        1/1     Running     0          22s
intrusion-detection-controller-bf9794dd7-5qxjs   1/1     Running     0          15m
intrusion-detection-es-job-installer-nfd7t       0/1     Completed   0          15m
```

### Above and beyond

- [Packet capture]({{site.baseurl}}/security/threat-detection-and-prevention/packetcapture)