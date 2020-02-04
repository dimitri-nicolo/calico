---
title: Tor-VPN feeds and Dashboard 
description: Detect and analyse malicious anonymization activity using Tor-VPN feeds.
canonical_url: /security/threat-detection-and-prevention/tor-vpn-feed-and-dashboard.md
---

### Big picture

Detect and analyse malicious anonymization activity using Tor-VPN feeds.

### Value

**Tor and VPN infrastructure** is used in enabling anonymous communication, where an attacker can leverage anonymity to scan, attack or compromise the target. It’s hard for network security teams to track malicious actors using such anonymization tools. Hence **Tor and VPN feeds** come into play where the feeds track all the tor bulk exit nodes as well as most of the anonymising VPN infrastructure on the internet. **The Tor-VPN Dashboard** helps network security teams to monitor and respond to any detected activity where they have a clusterwide view and granular control over logs which is critical in stopping the possible attack in early stages.

### Features

This how-to guide uses the following {{site.prodname}} features:
- **Tor-VPN Feeds**
- **Tor-VPN Dashboard**

### Concepts

#### About Tor and VPN threats

**Tor** is a popular anonymising network on the internet. It is also popular among the malicious actors, hacktivist groups, criminal enterprises as the infrastructure hides the real identity of an attacker carrying out malicious activities. To track down such attackers, tor historically was subject to investigation by various state level intelligence agencies from US and UK for criminal activities e.g. Silk Road marketplacei, Mirai botnet C&C. Though it’s not possible to completely de-anonymize the attacker. Hence **Tor bulk exit feed** came into existence to track all the tor exit IPs over the internet to know attackers using the tor infrastructure. 
Over the years, many Tor flaws became public and attackers evolved to leverage tor network with additional VPN layers. There are many individual VPN providers which have the anonymizing infrastructure. Attackers can use these new breed of VPN providers with existing options like tor to make sure of anonymity. To help security teams, the **EJR vpn feed** detects all the major private and public VPN providers on the internet.


#### Tor-VPN feed types

**Tor Bulk Exit feed**
The Tor Bulk Exit feed lists available tor exit nodes on the internet which are used by tor network. The list continuously updated and maintained by the Tor project. An attacker using Tor network, is likely to use one of the bulk exit nodes to connect to your infrastructure. The network security teams can detect such activity with Tor bulk exit feed and investigate as required. 

**EJR VPN feed**
In recent times it’s a trend to use multiple anonymization networks to hide real attacker identity. EJR VPN feed targets major VPN providers and their infrastructure used in anonymization activity over the internet. The feed comes up with bi-monthly updates which helps network security teams to stay on top of threats from such anonymizing infrastructure and detect them early in enumeration phase.

#### The {{site.prodname}} Tor-VPN dashboard

Tor-VPN dashboard helps network security teams to monitor and respond to any detected activity by Tor and VPN feeds. It provides a cluster context to the detection and shows multiple artifacts e.g. flow logs, filtering controls, a tag cloud and line graph to analyse the activity and respond faster.

### Before you begin...

#### Required

Privileges to manage GlobalThreatFeed.

#### Recommended

We recommend that you turn down the aggregation of flow logs sent to Elasticsearch for configuring threat feeds. If you do not adjust flow logs, Calico Enterprise aggregates over the external IPs for allowed traffic, and threat feed searches will not provide useful results (unless the traffic is denied by policy). Go to: [FelixConfiguration]({{site.baseurl}}/reference/resources/felixconfig) and set the field, **flowLogsFileAggregationKindForAllowed** to **1**.

### How to

In this section we will look at how to add Tor and VPN feeds to {{site.prodname}}. Installation process is straightforward as below.

1. Download tor-VPN manifests from [here]({{site.baseurl}}/manifests/threatdef/ejr-vpn.yaml) and [here]({{site.baseurl}}/manifests/threatdef/tor-exit-feed.yaml).
2. Add threat feed to the cluster.
   For EJR VPN,
   ```shell
   kubectl apply -f ejr-vpn.yaml
   ```
   For Tor Bulk Exit Feed,
   ```shell
   kubectl apply -f tor-exit-feed.yaml
   ```
3. Now, you can monitor the Dashboard for any malicious activity. The dashboard can be found at {{site.prodname}} manager, go to “kibana” and then go to “Dashboard”. Select “Tor-VPN Dashboard”.
4. Additionally, Feeds can be checked using following command.
   ```shell
   Kubectl get globalthreatfeeds 
   ```

### Tutorial

N/A

### Above and beyond

See [GlobalThreatFeed]({{site.baseurl}}/reference/resources/globalthreatfeed) resource definition for all configuration options.

