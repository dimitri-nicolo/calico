---
title: Extend FortiGate Firewalls to Kubernetes with Calico Enterprise
description: Enable FortiGate firewalls to control traffic from Kubernetes workloads using Calico Enterprise network policy
---

### Big picture

Use {{site.prodname}} network policy to control traffic from Kubernetes clusters in your FortiGate firewalls.

### Value

As platform and security engineers, you want your apps to securely communicate with the external world. But you also want to secure the network traffic from the Kubernetes clusters using your Fortigate firewalls. Using the Fortinet/{{site.prodname}} integration, security teams can retain firewall responsibility, secure traffic using {{site.prodname}} network policy, which frees up time for ITOps.

### Features

This how-to guide uses the following {{site.prodname}} features:

- {{site.prodname}}  **GlobalNetworkPolicy**
- {{site.prodname}}  **Tiers**

### Concepts

### How the integration works

The {{site.prodname}} integration controller (**tigera-firewall-controller**) lets you manage FortiGate firewall address group objects dynamically, based on {{site.prodname}} GlobalNetworkPolicy.

You determine the Kubernetes pods that you want to allow access outside the firewall, and create {{site.prodname}} global network policy using selectors that match those pods. After you deploy the tigera firewall controller in the Kubernetes cluster, you create a ConfigMap with the Fortinet firewall information. The {{site.prodname}} controller reads the ConfigMap, gets FortiGate firewall IP address and API token. It then populates the Kubernetes Node IPs of selector matching pods in FortiGate address group objects.


### Before you begin...

**Supported version**
- FortiGate version v6.2

**Required**

1. Experience with creating and administering Fortigate firewall policies.
1. Pull secret that you used during {{site.prodname}} installation

**Recommended**

Familiarity with [Calico tiered policy]({{site.baseurl}}/security/tiered-policy) and [Calico network policy]({{site.baseurl}}/security/calico-network-policy)

### How to

- [Configure FortiGate firewall to communicate with firewall controller](#configure-fortiGate-firewall-to-communicate-with-firewall-controller)

- [Create config map with FortiGate firewall information](#create-config-map-with-fortigate-firewall-information)

- [Deploy firewall controller in the Kubernetes cluster](#deploy-firewall-controller-in-the-kubernetes-cluster)

- [Create tier and global network policy](#create-tier-and-global-network-policy)

#### Configure FortiGate firewall to communicate with firewall controller


1. Determine and note the CIDR's or IP addresses of all Kubernetes nodes that can run the tigera-firewall-controller. This is required to whitelist the tigera-firewall-controller to access the FortiGate API.

2. Create an Admin profile  with read-write access to Address and Address Group Objects. For example: `calico_enterprise_api_user_profile`

3. Create a REST API Administrator and associate this user with the `calico_enterprise_api_user_profile` profile and add CIDR or IP address of your kubernetes cluster nodes as trusted hosts . For example:  `calico_enterprise_api_user`

4. Note the API key.

#### Create config map with FortiGate firewall information


1. Create a namespace for tigera-firewall-controller.

```
kubectl create namespace tigera-firewall-controller
```


2. Create config map with FortiGate firewall information

for example

```
kubectl -n tigera-firewall-controller create configmap  tigera-firewall-controller \
        --from-literal=tigera.firewall.policy.selector="projectcalico.org/tier == 'default'" \
        --from-literal=tigera.firewall.host=<IP-address-Of-FortiGate-firewall>
```

Description of ConfigMap section as follows

| Field                           | Enter values...                                                                                                                                                                                                                                        |
|---------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| tigera.firewall.host            | IP address of the FortiGate device.                                                                                                                                                                                                                    |
| tigera.firewall.policy.selector | The tier name with the global network policies with the Fortigate address group mappings.<br>For example, this selects the global network policies in the `default` tier:<br>`tigera.firewall.policy.selector: "projectcalico.org/tier == 'default'"   |


#### Deploy firewall controller in the Kubernetes cluster


1. Install your pull secret

```
   kubectl create secret generic tigera-pull-secret \
     --from-file=.dockerconfigjson=<path/to/pull/secret> \
     --type=kubernetes.io/dockerconfigjson -n tigera-firewall-controller
```

2. Install a secret to store the FortiGate API Key.

```
   kubectl create secret generic tigera-firewall-controller \
     --from-literal=apikey=<fortigate-api-secret> \
     -n tigera-firewall-controller
   ```
3. Apply manifest

```
   kubectl apply -f {{ "/manifests/fortinet.yaml" | absolute_url }}
```

#### Create tier and global network policy


1. Create a tier for organizing global network policies.

We recommend creating a separate [Calico tiered policy]({{site.baseurl}}/security/tiered-policies) for organizing all Fortigate firewall global network policies in a single location. (Use the Tier name as a selector in the ConfigMap for choosing global network policies for Fortigate firewalls.)

2. Create a GlobalNetworkPolicy for address group mappings.

For example, a GlobalNetworkPolicy can select a set of pods that require egress access to external workloads. In the following GlobalNetworkPolicy, the firewall controller creates an address group named, ‘default.production-microservice1’ in the Fortigate firewall. The members of ‘default.production-microservice1’ address group include IP addresses of nodes. Each node can host one or more pods whose label selector match with “"env == 'prod' && role == 'microservice1'". Each GlobalNetworkPolicy maps to an address group in FortiGate Firewall.

```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: default.production-microservice1
spec:
  selector: "env == 'prod' && role == 'microservice1'"
  types:
  - Egress
  egress:
  - action: Allow
```

### Verify integration is working


1. Log in to the Fortigate Firewall user interface.
2. Under Policy & Objects, click Addresses.
3. Verify that your Kubernetes-related address objects and address group objects are created with the following comments “Managed by Tigera {{site.prodname}}”.


### Above and beyond


- For additional Calico network policy features, see [Calico network policy]({{ site.baseurl }}/reference/resources/networkpolicy) and [Calico global network policy]({{ site.baseurl }}/reference/resources/globalnetworkpolicy)

