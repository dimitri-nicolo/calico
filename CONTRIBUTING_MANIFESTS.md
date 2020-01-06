# Contributing to manifests

This document explains how the static Kubernetes manifests are built on https://docs.tigera.io

## Background: Jekyll, liquid, and helm, oh my!

Pre-v2.4, our Manifests were templated using the [Liquid templating language](https://shopify.github.io/liquid/) - the available templating language for [Jekyll](https://jekyllrb.com/). This worked well for generating static manifests in the docs, but didn't work well if you wanted to generate manifests on the fly via CLI.

To make our manifests more CLI usable, there was a desire to convert our manifest templates into a Helm Chart. Unfortunately, Helm's templating language is [go templates](https://golang.org/pkg/text/template/), and Jekyll can't render go templates. So we taught it how!

- First, we converted all of our manifest templates 1:1 to go templates.

- Then, we registered a "custom tag" in Jekyll:

   ```
   {% helm %}
   datastore: kubernetes
   {% endhelm %}
   ```

- Any time Jekyll sees this, it calls our custom Jekyll plugin: `_plugins/helm.rb`.

- `helm.rb` executes Helm against our templates, passing everything between the tags to helm as a [values.yaml](https://helm.sh/docs/glossary/#values-values-files-values-yaml) file.

- The stdout of that execution (i.e. the rendered manifests) are then spat onto the page.

To summarize -  `helm.rb` is executing `helm` against our charts on behalf of the docs. The docs are no different than the standard Helm CLI user in that sense!

## Developer Use Cases

### Adding a new static manifest to the docs site

To add a new static manifest to the docs, create a new file that executes the helm plugin at the location where you want it to appear on the docs.

For example, if I want a file at `https://docs.tigera.io/master/manifests/policy-only-etcd-calico.yaml`:

```
tee ./master/manifests/policy-only-etcd-calico.yaml <<EOF
---
layout: null
---
{% helm %}
network: none
datastore: etcd
{% endhelm %}
EOF
```

### Only including certain files in a manifest

By default, helm renders **all** yaml template files. To only render certain files, pass `--execute` to the helm plugin:

```
{% helm --execute templates/calico-node.yaml --execute templates/calico-kube-controller.yaml %}
datastore: etcd
{% endhelm %}
```

>The path is relative to the Chart, so usually starts with `templates/`

**Why does every manifest call --execute?**

There are two reasons:

1. There are some resources that the helm manifests render that the docs shouldn't show, namely Secrets. Helm doesn't have an `--exclude` flag, so we have to `--execute` every file except for the ones that contain secrets.
2. TSEE is split into two Helm charts, but the Docs split EE into 4 manifests. To accomplish this, it uses `--execute` flags to only render some manifests from each chart. See more on this point [here](#)

### Choosing between helm charts

We've split Calico Enterprise into two helm charts:

- calico-enterprise-core
- calico-enterprise

By default, the helm plugin renders `calico-enterprise-core` (aka `calico`). To render the other chart, pass `calico-enterprise` as the first arg in the helm includer (before any `--execute`'s):

```
{% helm calico-enterprise --execute templates/manager.yaml %}
ping: pong
{% endhelm %}
```

### Rendering BYO elasticsearch manifests

The helm chart is "smart" when it comes to the elasticsearch connection:

- It assumes **Operator** Elasticsearch mode if **no** elasticsearch connection info / credentials are provided.
- It assumes **BYO** Elasticsearch mode if **all** elasticsearch connection info / credentials are provided.
- It **errors** if **some** elasticsearch connection info / credentials are provided.

It's a PITA to specify all credentials on every invocations when rendering the BYO manifests:

```yaml
{% helm calico-enterprise %}
elasticsearch:
  host: <>
  tls:
    ca: <>
  fluentd:
    password: <>
  manager:
    password: <>
  curator:
    password: <>
  compliance:
    controller:
      password: <>
    reporter:
      password: <>
    snapshotter:
      password: <>
    server:
      password: <>
    benchmarker:
      password: <>
  intrusionDetection:
    password: <>
  elasticInstaller:
    password: <>
{% endhelm %}
```

So we created a special `secure-es` flag which instructs `helm.rb` to pass that^ to helm:

```
{% helm calico-enterprise secure-es %}
createCustomResources: false
{% endhelm %}
```

Much simpler!

### Adding a new Resource

#### 1. Figure out which manifest it belongs in

To add a new resource, ask yourself if there's an existing rendered manifest this belongs in. The following information explains how dependencies are handled in the install procedure and should help you to identify which manifest to add it to:

[chart] charts/calico-enterprise-core:

- calico.yaml
  - gets nodes "ready" by installing networking
  - installs cnx-apiserver (for application of Calico resources via kubectl)
  - creates CRDs

[chart] charts/calico-enterprise:

1. operator.yaml
   - installs 3rd party CRDs
1. monitor-calico.yaml
   - installs 3rd party CRs
1. cnx.yaml
   - installs all the rest of the sweet sweet EE sauce

#### 2. Figure out which file it belongs in

Thanks to templating, we can split up our resources into as many template files as we want for organization.

Use your discretion here. Some tips:

- Group resources together in the same file based off the "component", not by the resource type.
  - Example: Don’t create a yaml file for "configs". Create a yaml file for "manager", and include it’s deployment, rbac, etc.
- If a yaml gets gets too large, create a directory for the component, and split it into smaller yaml files.
  - Example: Compliance

#### 3. If you've added a new file, don't forget to --execute it where necessary

Since most rendered docs manifests explicitly call `--execute` when rendering the charts, they will not pick up a new file automatically.

Make sure to `--execute` your new file in all static manifests that you need it to show in.

## Manifest Templating with Helm

The values used for templating the manifests are technically a public API. Adding new values or modifying existing ones should be done with care as they will impact customers.

#### Helm Values

Helm values for Tigera are dynamically generated by parsing versioning information in `_data/versions.yml`, image names in `_config.yml`, and other settings from `_plugins/<version>/values.rb`. You can generate the `values.yaml` yourself by running `make values.yaml`.

You can modify the values by making changes to `_plugins/<version>/values.rb`.
