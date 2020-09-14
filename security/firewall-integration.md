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

You determine the Kubernetes pods that you want to allow access outside the firewall, and create {{site.prodname}} global network policy using selectors that match those pods. After you deploy the tigera firewall controller in the Kubernetes cluster, you create a ConfigMap with the Fortinet firewall information. The {{site.prodname}} controller reads the ConfigMap, gets FortiGate firewall IP address, API token and source IP address selection, it can be either `node` or `pod`.
- In your kubernetes cluster, if pods IP addresses are routable and address selection is `pod`, then it populates the Kubernetes pod IPs of selector matching pods in FortiGate address group objects or 
- If source address selection is `node`, then populates the kubernetes node IPs of selector matching pods in Fortigate address group objects.


### Before you begin...

**Supported version**
- FortiGate version v6.2
- FortiManager version v6.3

**Required**

1. Experience with creating and administering FortiGate/FortiManager firewall policies.
1. Pull secret that you used during {{site.prodname}} installation

**Recommended**

Familiarity with [Calico tiered policy]({{site.baseurl}}/security/tiered-policy) and [Calico network policy]({{site.baseurl}}/security/calico-network-policy)


### How to

- [Configure FortiGate firewall to communicate with firewall controller](#configure-fortiGate-firewall-to-communicate-with-firewall-controller)

- [Configure FortiManager to communicate with firewall controller](#configure-fortiManager-to-communicate-with-firewall-controller)

- [Create config map for address selection in Firewall Controller](#create-config-map-for-address-selection-in-firewall-controller)

- [Install FortiGate ApiKey and FortiManager password as Secrets](#install-fortigate-apikey-and-fortimanager-password-as-secrets)

- [Deploy firewall controller in the Kubernetes cluster](#deploy-firewall-controller-in-the-kubernetes-cluster)

- [Create tier and global network policy](#create-tier-and-global-network-policy)

#### Configure FortiGate firewall to communicate with firewall controller

1. Determine and note the CIDR's or IP addresses of all Kubernetes nodes that can run the tigera-firewall-controller. This is required to explicitly allow the tigera-firewall-controller to access the FortiGate API.

2. Create an Admin profile  with read-write access to Address and Address Group Objects. For example: `tigera_api_user_profile`

3. Create a REST API Administrator and associate this user with the `tigera_api_user_profile` profile and add CIDR or IP address of your kubernetes cluster nodes as trusted hosts . For example:  `calico_enterprise_api_user`

4. Note the API key.

#### Configure FortiManager to communicate with firewall controller

1. Determine and note the CIDR's or IP addresses of all Kubernetes nodes that can run the tigera-firewall-controller. This is required to explicitly allow the tigera-firewall-controller to access the FortiManager API.

2. From system settings, create an Admin profile with Read-Write access for `Policy & Objects`. For example: `tigera_api_user_profile`

3. Create a JSON API administrator and associate this user with the `tigera_api_user_profile` profile and add CIDR or IP address of your kubernetes cluster nodes as `Trusted Hosts`.

4. Note username and password.

#### Create config map for address selection in Firewall Controller

1. Create a namespace for tigera-firewall-controller.

   ```
   kubectl create namespace tigera-firewall-controller
   ```

2. Create config map with FortiGate firewall information

   for example

   ```
   kubectl -n tigera-firewall-controller create configmap  tigera-firewall-controller \
   --from-literal=tigera.firewall.policy.selector="projectcalico.org/tier == 'default'" \
   --from-literal=tigera.firewall.addressSelection="node"
   ```

    Description of ConfigMap section as follows

    | Field                            | Enter values...                                                                                                                                                                                                                                                                     |
    |----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
    | tigera.firewall.policy.selector  | The tier name with the global network policies with the Fortigate address group mappings.<br>For example, this selects the global network policies in the `default` tier:<br>`tigera.firewall.policy.selector: "projectcalico.org/tier == 'default'"                                |
    | tigera.firewall.addressSelection | The addressSelection for outbound traffic leaving the cluster.<br>For example, if outgoingNat is enabled in cluster and compute Node IP address is used "tigera.firewall.addressSelection == `node` or <br> If pod IP address used then "tigera.firewall.addressSelection == `pod`" |

#### Create config map with FortiGate and FortiManager Information

1. In this ConfigMap manifest [file]({{site.baseurl}}/manifests/fortinet-device-configmap.yaml), add your FortiGate firewall Information  under data section `tigera.firewall.fortigate` as below

for example,

```
- name: prod-eastcoast-1
  ip: 1.2.3.1
  apikey:
    secretKeyRef:
      name: fortigate-east1
      key: apikey-fortigate-east1
- name: prod-eastcoast-2
  ip: 1.2.3.2
  apikey:
    secretKeyRef:
      name: fortigate-east2
      key: apikey-fortigate-east2
```

2. In this ConfigMap manifest [file]({{site.baseurl}}/manifests/fortinet-device-configmap.yaml), add your FortiManager Information  under data section `tigera.firewall.fortimgr` as below

for example

```
- name: prod-east1
  ip: 1.2.4.1
  username: api_user
  adom: root
  password:
    secretKeyRef:
      name: fortimgr-east1
      key: pwd-fortimgr-east1
```
3. Apply modified manifest

   ```
   kubectl apply -f {{ "/manifests/fortinet-device-configmap.yaml" | absolute_url }}
   ```

check, Reference section to know more details about field descriptions in the configmap.

#### Install FortiGate ApiKey and FortiManager password as Secrets

1. Store each FortiGate API key as Secret in tigera-firewall-controller namespace.

for example, In the above config map for FortiGate device prod-east1, store its ApiKey as a secret name as `fortigate-east1`, with key as `apikey-fortigate-east1` 

```
kubectl create secret generic fortigate-east1 \
-n tigera-firewall-controller \
--from-literal=apikey-fortigate-east1=<fortigate-api-secret>
```

2. Store each FortiManager Password as Secret in tigera-firewall-controller namespace.

for example, In the above config map for FortiMgr `prod-east1`, store its Password as a secret name as `fortimgr-east1`, with key as `pwd-fortimgr-east1` 

```
kubectl create secret generic fortimgr-east1 \
-n tigera-firewall-controller \
--from-literal=pwd-fortimgr-east1=<fortimgr-password>
```

#### Deploy firewall controller in the Kubernetes cluster

1. Install your pull secret

   ```
   kubectl create secret generic tigera-pull-secret \
   --from-file=.dockerconfigjson=<path/to/pull/secret> \
   --type=kubernetes.io/dockerconfigjson -n tigera-firewall-controller
   ```

2. Apply manifest

   ```
   kubectl apply -f {{ "/manifests/fortinet.yaml" | absolute_url }}
   ```

#### Create tier and global network policy

1. Create a tier for organizing global network policies.

    We recommend creating a separate [Calico tiered policy]({{site.baseurl}}/security/tiered-policy) for organizing all Fortigate firewall global network policies in a single location. (Use the Tier name as a selector in the ConfigMap for choosing global network policies for Fortigate firewalls.)

2. Create a GlobalNetworkPolicy for address group mappings.

    For example, a GlobalNetworkPolicy can select a set of pods that require egress access to external workloads. In the following GlobalNetworkPolicy, the firewall controller creates an address group named, `default.production-microservice1` in the Fortigate firewall. The members of `default.production-microservice1` address group include IP addresses of nodes. Each node can host one or more pods whose label selector match with `env == 'prod' && role == 'microservice1'`. Each GlobalNetworkPolicy maps to an address group in FortiGate Firewall.

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

1. Log in to the FortiGate's firewall user interface.
2. Under Policy & Objects, click Addresses.
3. Verify that your Kubernetes-related address objects and address group objects are created with the following comments "Managed by Tigera {{site.prodname}}".
4. If you have any FortiManager configured to work with firewall-controller, log in to each FortiManager's user interface with correct ADOM.
5. Click Policy & Objects, under Object Configuration choose Addresses.
6. Verify that your Kubernetes-related address objects and address group objects are created with the following comments "Managed by Tigera {{site.prodname}}".

### Reference

More information of ConfigMap manifest [file]({{site.baseurl}}/manifests/fortinet-device-configmap.yaml) which is used to input FortiGate and FortiManager device information to firewall-controller

Description of FortiGate device as follows

| Field                    | Description                                                                 |
|--------------------------|-----------------------------------------------------------------------------|
| name                     | FortiGate device name                                                       |
| ip                       | FortiGate Management Ip address                                             |
| apikey                   | Secret in tigera-firewall-controller namespace, to store FortiGate's APIKey |
| apikey.secretKeyRef.name | Name of the secret to store APIKey.                                         |
| apikey.secretKeyRef.key  | Key name in the secret, which stores APIKey                                 |


Description of FortiManager device as follows

| Field                      | Description                                                                    |
|----------------------------|--------------------------------------------------------------------------------|
| name                       | FortiManager device name                                                       |
| ip                         | FortiManager Management Ip address                                             |
| adom                       | FortiManager ADOM name to manage kubernetes cluster.                           |
| username                   | JSON api access account name to Read/Write FortiManager address objects.       |
| password                   | Secret in tigera-firewall-controller namespace, to store FortiManager password |
| password.secretKeyRef.name | Name of the secret to store password.                                          |
| password.secretKeyRef.key  | Key name in the secret, which stores password.                                 |


### Above and beyond

- For additional Calico network policy features, see [Calico network policy]({{ site.baseurl }}/reference/resources/networkpolicy) and [Calico global network policy]({{ site.baseurl }}/reference/resources/globalnetworkpolicy)
