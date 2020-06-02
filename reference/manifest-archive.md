---
title: Install a previous release of Calico Enterprise
description: Install a previous patch release of Calico Enterprise using archive.
---

### Big picture

Install a previous patch release of {{ site.prodname }} starting with the 3.0 release.


### How to

1. From [release-notes]({{site.baseurl}}/release-notes/), download the archive for desired version of {{ site.prodname }}.

1. Follow the docs for the relevant version, and use the appropriate manifest from the archive wherever you are prompted to download a manifest.

**Note the following:** 

- The folder structure of the archive reflects the download url paths present in the docs. For example, from the docs for OpenShift installation, we have the following command:

    ```bash
    curl {{ "/manifests/ocp/crds/01-crd-alertmanager.yaml" | absolute_url }} -o manifests/01-crd-alertmanager.yaml
    ```

   To access the relevant manifest, navigate to the extracted archive as below:

    ```
     manifests (folder) -> ocp (folder) -> 01-crd-alertmanager.yaml (required file)
    ``` 

- For the following command:

    ```bash
    kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
    ```

   it would be

    ``` bash
    cd manifests && kubectl create -f tigera-operator.yaml
    ```
- (Optional) If you are using the OCP platform, the archive contains a bash script `collect-ocp-manifests.sh`, which collects all the OpenShift-related manifests from the archive to a separate `ocp-manifests` directory for your convenience. 

