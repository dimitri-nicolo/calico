---
title: Installing Calico Enterprise using Helm
description: Use Helm application package manager to install Calico Enterprise. 
---

### Big Picture

Install {{ site.prodname }} in Kubernetes using Helm.

### Before you begin

- Ensure that you have Tiller **v2.14** running, and your local helm CLI is configured to speak to it.

- Ensure that you have the [credentials for the Tigera private registry and a license key]({{site.baseurl}}/getting-started/calico-enterprise)

### How to

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Acquire the Helm chart

   ```
   curl -O -L https://s3.amazonaws.com/tigera-public/ee/charts/tigera-operator-{% include chart_version_name %}.tgz
   ```

1. Install the chart, passing in your image pull secrets:

   ```
   helm install ./tigera-secure-ee-core-{% include chart_version_name %}.tgz \
     --name {{ site.prodnamedash }} \
     --set-file imagePullSecrets.tigera-pull-secret=./config.json
   ```

1. Wait for the apiserver to become available:

   ```
   kubectl wait --for condition=available tigerastatus installation apiserver
   ```

1. Install your {{ site.prodname }} license:

   ```
   kubectl apply -f ./license.yaml
   ```

1. Apply the following manifest to set network policy that secures access to {{ site.prodname }}:

   ```
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

### Above and beyond

- [Configure access to Calico Enterprise Manager]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Get started with Kubernetes network policy]({{site.baseurl}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.baseurl}}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{site.baseurl}}/security/kubernetes-default-deny)
