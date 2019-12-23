---
title: Installing Tigera Secure EE for policy (advanced)
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/other
---

## About installing {{site.tseeprodname}} for policy

You can also use {{site.tseeprodname}} just for policy enforcement and achieve networking
with another solution, such as static routes or a Kubernetes cloud provider integration.

To install {{site.tseeprodname}} in this mode using the Kubernetes API datastore,
complete the following steps.

## Before you begin

- Ensure that you have a Kubernetes cluster that meets the {{site.tseeprodname}}
  [system requirements](../requirements). If you don't, follow the steps in
  [Using kubeadm to create a cluster](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

- Ensure that you have the [private registry credentials](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

{% include {{page.version}}/load-docker-intro.md %}

{% include {{page.version}}/load-docker-our-reg.md yaml="calico" %}

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

## <a name="install-cnx"></a>Installing {{site.tseeprodname}} for policy

1. Ensure that you have a Kubernetes cluster that meets the
   {{site.tseeprodname}} [system requirements](../requirements). If you don't,
   follow the steps in [Using kubeadm to create a cluster](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.tseeprodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
   ```
   > **Note**: You can also
   > [view the manifest in your browser](hosted/rbac-kdd.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Ensure that the Kubernetes controller manager has the following flags
   set: <br>
   `--cluster-cidr=192.168.0.0/16` and `--allocate-node-cidrs=true`.

   > **Tip**: On kubeadm, you can pass `--pod-network-cidr=192.168.0.0/16`
   > to kubeadm to set both Kubernetes controller flags.
   {: .alert .alert-success}

1. Download the {{site.tseeprodname}} policy-only manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only/1.7/calico.yaml \
   -O
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. If your cluster contains more than 50 nodes:

   - In the `ConfigMap` named `calico-config`, locate the `typha_service_name`,
     delete the `none` value, and replace it with `calico-typha`.

   - Modify the replica count in the`Deployment` named `calico-typha`
     to the desired number of replicas.

     ```
     apiVersion: apps/v1beta1
     kind: Deployment
     metadata:
       name: calico-typha
       ...
     spec:
       ...
       replicas: <number of replicas>
     ```

     We recommend at least one replica for every 200 nodes and no more than
     20 replicas. In production, we recommend a minimum of three replicas to reduce
     the impact of rolling upgrades and failures.

     > **Tip**: If you set `typha_service_name` without increasing the replica
     > count from its default of `0` Felix will try to connect to Typha, find no
     > Typha instances to connect to, and fail to start.
     {: .alert .alert-success}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md %}

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" %}

{% include {{page.version}}/cnx-monitor-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
