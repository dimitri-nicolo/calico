---
title: Getting started with Calico Enterprise
canonical_url: https://docs.tigera.io/v2.3/getting-started/
---

## Obtain the private registry credentials

Contact your Tigera support representative to obtain a Docker configuration file
in JSON format. When you open the file, it should look something like the following.

```json
{
  "auths": {
    "quay.io": {
      "auth": "<ROBOT-TOKEN-VALUE>",
      "email": ""
    }
  }
}
```

The file should be named `config.json`. It contains a robot account token that will allow you to retrieve the {{site.prodname}} images from the private Tigera repository.

## Obtain a license key

Contact your Tigera support representative to obtain a license key in YAML format.
When you open the file, it should look something like the following.

```yaml
apiVersion: projectcalico.org/v3
kind: LicenseKey
metadata:
  creationTimestamp: null
  name: default
spec:
  certificate: |
    -----BEGIN CERTIFICATE-----
    MII...n5
    -----END CERTIFICATE-----
  token: eyJ...zaQ
```

The file should be named `<customer-name>-license.yaml`. For example, if your customer name
was Awesome Corp, the file would be named `awesome-corp-license.yaml`.

## Get started

<div class="row">
  <div class="col-xs-6 col-md-3" style="text-align:center">
    <a href="/{{page.version}}/getting-started/kubernetes/" class="thumbnail">
      <img src="{{site.baseurl}}/images/kubernetes_logo.svg" alt="Kubernetes" width="87%">
    </a>
    Kubernetes
  </div>
  <div class="col-xs-6 col-md-3" style="text-align:center">
    <a href="/{{page.version}}/getting-started/kubernetes/installation/eks" class="thumbnail">
      <img src="{{site.baseurl}}/images/icon-aws-amazon-eks.svg" alt="Amazon EKS" width="75%">
    </a>
    Amazon EKS
  </div>
  <div class="col-xs-6 col-md-3" style="text-align:center">
    <a href="/{{page.version}}/getting-started/kubernetes/installation/aks" class="thumbnail">
      <img src="{{site.baseurl}}/images/icon-azure-kubernetes-services.svg" alt="Azure AKS" width="85%">
    </a>
    Azure AKS 
  </div>
  <div class="col-xs-6 col-md-3" style="text-align:center">
    <a href="/{{page.version}}/getting-started/kubernetes/installation/docker-ee" class="thumbnail">
      <img src="{{site.baseurl}}/images/Docker-R-Logo-08-2018-Monochomatic-RGB_Vertical-x3.jpg" alt="Docker Enterprise" width="98%">
    </a>
    Docker Enterprise
  </div>
  <div class="col-xs-6 col-md-3" style="text-align:center">
    <a href="/{{page.version}}/getting-started/openshift/installation/" class="thumbnail">
      <img src="{{site.baseurl}}/images/OpenShift-LogoType.svg" alt="OpenShift" width="80%">
    </a>
    OpenShift
  </div>
</div>
