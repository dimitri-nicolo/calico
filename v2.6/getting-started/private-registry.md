---
title: Installing from a private registry
redirect_from: latest/getting-started/private-registry
---

{% assign operator = site.data.versions[page.version].first.tigera-operator %}

### Big picture

Move Tigera container images to a private registry and configure {{ site.prodname }} to pull images from it.

### Value

Install Tigera Secure in clusters where pulling from third party private repos is not an option, such as airgapped clusters, or clusters with bandwidth constraints or security constraints.

### Concepts

A **container image registry** (often referred to as a **registry**) is a service where container images are pushed to, stored, and pulled from. A registry is said to be "private" if it requires users authenticate before accessing images.

An **image pull secret** is used in Kubernetes to deploy container images from a private container image registry.

### Before you begin...

- Configure pull access to your private registry
- [Configure pull access to Tigera's private container registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials).

### How to

- [Push {{ site.prodname }} images to your private registry](#push-{{ site.prodname | slugify }}-images-to-your-private-registry)
- [Run the operator using images from your private registry](#run-the-operator-using-images-from-your-private-registry)
- [Configure the operator to use images from your private registry](#configure-the-operator-to-use-images-from-your-private-registry)

#### Push {{ site.prodname }} images to your private registry

In order to install images from your private registry, you must first pull the images from Tigera's registry, re-tag them with your own registry, and then push the newly tagged images to your own registry.

1. Use the following commands to pull the required {{site.prodname}} images.

   ```bash
   docker pull {{ operator.registry }}/{{ operator.image }}:{{ operator.version }}
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image -%}
   {% if component[1].registry %}{% assign registry = component[1].registry | append: "/" %}{% else %}{% assign registry = page.registry -%} {% endif -%}
   docker pull {{ registry }}{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```

1. Retag the images with the name of your private registry `$PRIVATE_REGISTRY`.

   ```bash
   docker tag {{ operator.registry }}/{{ operator.image }}:{{ operator.version }} $PRIVATE_REGISTRY/{{ operator.image }}:{{ operator.version }}
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image -%}
   {% if component[1].registry %}{% assign registry = component[1].registry | append: "/" %}{% else %}{% assign registry = page.registry -%} {% endif -%}
   docker tag {{ registry }}{{ component[1].image }}:{{component[1].version}} $PRIVATE_REGISTRY/{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```

1. Push the images to your private registry.

   ```bash
   docker push $PRIVATE_REGISTRY/{{ operator.image }}:{{ operator.version }}
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image -%}
   docker push $PRIVATE_REGISTRY/{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```

   > **Important**: Do not push the private {{site.prodname}} images to a public registry.
   {: .alert .alert-danger}

#### Run the operator using images from your private registry

Before applying `tigera-operator.yaml`, modify registry references to use your custom registry:

```
{% if page.registry != "quay.io/" -%}
sed -ie "s?{{ page.registry }}?$PRIVATE_REGISTRY?g" tigera-operator.yaml
{% endif -%}
sed -ie "s?quay.io?$PRIVATE_REGISTRY?g" tigera-operator.yaml
```
{% comment %} The second 'sed' should be removed once operator launches Prometheus & Alertmanager {% endcomment %}

Before applying `custom-resources.yaml`, modify registry references to use your custom registry:

```
sed -ie "s?quay.io?$PRIVATE_REGISTRY?g" custom-resources.yaml
```
{% comment %} This step should be removed once operator launches Prometheus & Alertmanager {% endcomment %}

#### Configure the operator to use images from your private registry.

Set the `spec.registry` field of your Installation resource to the name of your custom registry. For example:

<pre>
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  variant: TigeraSecureEnterprise
  imagePullSecrets:
    - name: tigera-pull-secret
  <b>registry: myregistry.com</b>
</pre>

>**Note:** See [the Installation resource reference page](/{{page.version}}/reference/installation/api) for more information on the `imagePullSecrets` and `registry` fields.
{: .alert .alert-info }
