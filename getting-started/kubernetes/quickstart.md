---
title: Quickstart for Calico Enterprise on Kubernetes
description: Install Calico Enterprise on a single-host Kubernetes cluster.
canonical_url: '/getting-started/kubernetes/index'
---

### Big picture

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}} in approximately 15 minutes.

### Value

You can use this quickstart to quickly and easily try {{side.prodname}} features.

### How to

- [Host requirements](#host-requirements)
- [Install Kubernetes](#install-kubernetes)
- [Install {{site.prodname}}](#install-calico-enterprise)
- [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
- [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

#### Host requirements

This article requires a Linux host that meets the following requirements.

- AMD64 processor
- 2CPU
- 12GB RAM
- 50GB free disk space
- Ubuntu Server 16.04
- Internet access
- [Sufficient virtual memory](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html){:target="_blank"}

#### Install Kubernetes

1. [Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

1. As a regular user with sudo privileges, open a terminal on the host that you installed kubeadm on.

1. Initialize the master using the following command.

   ```bash
   sudo kubeadm init --pod-network-cidr=192.168.0.0/16 \
   --apiserver-cert-extra-sans=127.0.0.1
   ```

   > **Note**: If 192.168.0.0/16 is already in use within your network you must select a different pod network
   > CIDR, replacing 192.168.0.0/16 in the above command as well as in any manifests applied below.
   {: .alert .alert-info}

1. Execute the following commands to configure kubectl (also returned by `kubeadm init`).

   ```bash
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```

1. Remove master taint in order to allow kubernetes to schedule pods on the master node.

   ```bash
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```
   kubectl create -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   You can now monitor progress with the following command:

   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

#### Secure {{site.prodname}} with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```
#### Log in to the {{site.prodname}} Manager UI

```
kubectl port-forward -n tigera-manager svc/tigera-manager 9443 &
```

Sign in by navigating to https://localhost:9443 and login.

##### Kibana authentication

Connect to Kibana with the `elastic` username. Use the following command to decode the password:

```
kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o go-template='{{.data.elastic | base64decode}}' && echo
```

### Above and beyond

- [Configure access to the manager UI]({{site.baseurl}}/getting-started/access-the-manager)
- [Get started with Kubernetes network policy]({{site.baseurl}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.baseurl}}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{site.baseurl}}/security/kubernetes-default-deny)
