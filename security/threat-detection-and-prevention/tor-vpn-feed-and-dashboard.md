---
title: Tor-VPN feeds and Dashboard 
description: Tor-VPN feeds and dashboard to detect and analyse the malicious masking activity.
canonical_url: /security/threat-detection-and-prevention/tor-vpn-feed-and-dashboard.md
---

### Big picture

Use Tor-VPN feeds and dashboard to detect and analyse the malicious masking activity

### Value

Once {{site.prodname}} is deployed on clusters and workloads are spun up. Network security team would want to know if any one of the workloads is compromised and talking to suspicious IPs.
An attacker likes to hide his real identity behind the tor network or VPN infrastructure.
Hence if the network security team detects tor network or VPN flows in the cluster, it could be a good indicator of malicious activity.
With  {{site.prodname}}, we are providing Tor-VPN feeds which will help in detecting malicious masking activity. At the same time we have created a Tor-VPN Dashboard which will provide granular view over such activities clusterwide so that network security teams can investigate and efficiently respond to detected threats.

### Features

This how-to guide uses the following {{site.prodname}} features:
- **Tor-VPN Feeds
- **Tor-VPN Dashboard

### Concepts

#### Tor-VPN Feeds: when and why

The best time to use Tor-VPN feeds and Dashboard would be when the workloads are deployed on the cluster and network security teams have started day to day monitoring.
Additionally when, 
- Network security administrators want to secure and know any masking activity in their clusters
- Organisation who has suffered an attack previously and want to know if there is any presence of malicious actors who may use VPN or Tor networks to access the cluster.

#### Tor-VPN feeds

Tor is a popular masking network on the internet, attackers commonly use to enumerate, attack and access backdoors on targets conveniently hiding it’s true identity. At the same time, using tor attacker has resources from the dark web to further his motive. The Tor and VPN feeds monitor and detect all the such tor exit nodes on the internet, if attacker uses on of the tor exit nodes then Tor bulk exit feed will be able to detect such activity

Apart from tor, there are many many individual and paid VPN providers that attacker can leverage. With EJR vpn feed, we can detect all the major VPN providers on the internet.

#### Tor-VPN Dashboard

Tor-VPN dashboard makes it easy to analyse and respond to any detected activity from tor and EJR vpn feeds. Tor-VPN dashboard can pinpoint activity to a workload and show rate of detection in the cluster with options to filter artifacts related to the malicious activity. The dashboard shows flow logs, filtering controls, a tag cloud and line graph to analyse the activity. 
The dashboard can be found at {{site.prodname}} manager, go to “kibana” and then go to “Dashboard”. Select “Tigera Secure EE Tor-VPN Logs”

### Before you begin...

#### Required

Privileges to manage GlobalThreatFeed.

#### Recommended

We recommend that you turn down the aggregation of flow logs sent to Elasticsearch for configuring threat feeds. If you do not adjust flow logs, Calico Enterprise aggregates over the external IPs for allowed traffic, and threat feed searches will not provide useful results (unless the traffic is denied by policy). Go to: [FelixConfiguration]({{site.baseurl}}/reference/resources/felixconfig) and set the field, **flowLogsFileAggregationKindForAllowed** to **1**.

### How to

In this section we will look at how to add Tor and VPN feeds to  {{site.prodname}}. Installation process is straightforward as below.

1. Download tor-VPN manifests from [here]({{site.baseurl}}/ref).  
2. Add threat feed to the cluster
   For EJR VPN,
   ```shell
   kubectl apply -f ejr-vpn.yaml
   ```
   For Tor Bulk Exit Feed,
   ```shell
   kubectl apply -f tor-exit-feed.yaml
   ```
3. Now, you can monitor the Dashboard for any malicious activity. The dashboard can be found at {{site.prodname}} manager, go to “kibana” and then go to “Dashboard”. Select “Tor-VPN Dashboard”
4. Additionally, Feeds can be checked using following command
   ```shell
   Kubectl get globalthreatfeeds 
   ```
If you need additional information w.r.t. blocking the IPs from these feed, please go through the concept documentation [here]({{site.baseurl}}/ref).

### Tutorial

N/A

### Above and beyond

#### Threat Feed details

- **Tor Bulk Exit feed**
Tor Bulk Exit feed is the list of all available tor exit nodes on the internet. It is continuously updated and maintained by the Tor project. An attacker wanting to mask his IP with tor network is likely to use one of the tor bulk exit nodes from this feed, hence it’s easy to detect any tor related activity in your cluster. 

- **EJR VPN feed**
EJR VPN feed targets major VPN providers and datacenters used in masking activity. The feed comes up with bi-monthly updates.


See [GlobalThreatFeed]({{site.baseurl}}/reference/resources/globalthreatfeed) resource definition for all configuration options.

