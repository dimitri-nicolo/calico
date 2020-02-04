---
title: Tor-VPN feeds and Dashboard 
description: Detect and analyse malicious masking activity using Tor-VPN feeds.
canonical_url: /security/threat-detection-and-prevention/tor-vpn-feed-and-dashboard.md
---

### Big picture

Detect and analyse malicious masking activity using Tor-VPN feeds.

### Value

Tor and VPN are both tools for privacy and anonymity, but they are also popular among malicious actors. Network security teams need visibility into workloads that are scanned, attacked, or compromised by attacker hiding behind such tools. {{site.prodname}} provides **Tor-VPN** feeds for detecting malicious masking activity, including a **Tor-VPN Dashboard** with a granular view of Tor-VPN activities clusterwide.

### Features

This how-to guide uses the following {{site.prodname}} features:
- **Tor-VPN Feeds**
- **Tor-VPN Dashboard**

### Concepts

#### About Tor and VPN threats

**Tor** is a popular masking network on the internet used by attackers. It is popular among the malicious actors because the infrastructure hides their real identity.  Attackers can scan, attack, compromise targets, and access backdoors with anonymity. Because infrastructures like Tor have resources from the dark web, this further incentivizes attackers. 

Along with tor, there are many individual and paid VPN providers that attackers can leverage. With EJR vpn feed, {{site.prodname}} can accurately detect all the major private and public VPN providers on the internet.

#### Tor-VPN feed types

**Tor Bulk Exit feed**
The Tor Bulk Exit feed lists all available tor exit nodes on the internet. It is continuously updated and maintained by the Tor project. An attacker wanting to mask his IP with tor network is likely to use one of the tor bulk exit nodes from this feed, so it’s easy to detect any tor related activity in your cluster. 

**EJR VPN feed**
EJR VPN feed targets major VPN providers and data centers used in masking activity. The feed comes up with bi-monthly updates.

#### The {{site.prodname}} Tor-VPN dashboard

Using the Tor-VPN dashboard, you can pinpoint activity to a workload within a cluster context, and show rate of detection in the cluster, and filter artifacts related to the malicious activity. The dashboard shows flow logs, filtering controls, a tag cloud, and line graph to analyse the activity.

### Before you begin...

#### Required

Privileges to manage GlobalThreatFeed.

#### Recommended

We recommend that you turn down the aggregation of flow logs sent to Elasticsearch for configuring threat feeds. If you do not adjust flow logs, Calico Enterprise aggregates over the external IPs for allowed traffic, and threat feed searches will not provide useful results (unless the traffic is denied by policy). Go to: [FelixConfiguration]({{site.baseurl}}/reference/resources/felixconfig) and set the field, **flowLogsFileAggregationKindForAllowed** to **1**.

### How to

In this section we will look at how to add Tor and VPN feeds to  {{site.prodname}}. Installation process is straightforward as below.

1. Download tor-VPN manifests from [here]({{site.baseurl}}/manifests/threatdef/ejr-vpn.yaml) and [here]({{site.baseurl}}/manifests/threatdef/tor-exit-feed.yaml).  
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
If you need additional information w.r.t. blocking the feed IPs, please go through the concept documentation [here]({{site.baseurl}}/security/threat-detection-and-prevention/suspicious-ips).

### Tutorial

N/A

### Above and beyond

See [GlobalThreatFeed]({{site.baseurl}}/reference/resources/globalthreatfeed) resource definition for all configuration options.

