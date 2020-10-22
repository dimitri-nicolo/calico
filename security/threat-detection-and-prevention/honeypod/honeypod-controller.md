---
title: Monitor honeypods
description: Monitor honeypod behavior to gain insight on what attackers are doing.
canonical_url: /security/threat-detection-and-prevention/honeypod/honeypod-controller
---

### Big picture

Monitor honeypod behavior to gain insight on what attackers are doing using packet-level inspection.

### Value

Adding monitoring to honeypods extends your ability to thwart activities including
generating alerts if traffic to honeypods match any Intrusion Detection System(Snort) signatures, and scanning traffic if activities are detected.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **PacketCapture** with **Honeypod controller**

### Concepts

#### Monitoring implementation

Monitoring honeypods is done using a controller that periodically polls selected honeypods for suspicious activity and scans its traffic. Alerts are generated in the Events tab of {{site.prodname}} Manager UI.

The controller leverages the following:

- [Packet capture feature]({{site.baseurl}}/security/threat-detection-and-prevention/packetcapture) to collect honeypod traffic in clusters.
- Open source {% include open-new-window.html text='Snort' url='https://www.snort.org/' %} to scan honeypod traffic.

### Before you begin

#### Required

[Honeypods are configured]({{site.baseurl}}/security/threat-detection-and-prevention/honeypod/honeypods) for clusters, and alerts are generated when the honeypods are accessed.

### How To

  - [Add honeypod controller to cluster](#add-honeypod-controller-to-cluster)
  - [Troubleshooting](#troubleshooting)

#### Add honeypod controller to cluster

Add the honeypod controller to each cluster configured for honeypods using the following command:

```shell
kubectl apply -f {{ "/manifests/threatdef/honeypod/controller.yaml" | absolute_url }} 
```

#### Troubleshooting

To troubleshoot honeypod controller, see [Troubleshooting]({{site.baseurl}}/maintenance/troubleshoot/troubleshooting)