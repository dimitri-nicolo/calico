---
title: Enabling application layer policy
canonical_url: https://docs.tigera.io/v2.2/getting-started/kubernetes/installation/app-layer-policy
---

## About enabling application layer policy

Application layer policy for {{site.prodname}} allows you to write policies that
enforce against application layer attributes like HTTP methods or paths as well as
against cryptographically secure identities.

> **Note**: Application layer policy is a beta feature of this release and not
> recommended for production use.
{: .alert .alert-info}

Support for application layer policy is not enabled by default in
{{site.prodname}} installs, since it requires extra CPU and memory resources to
operate.

## Enabling application layer policy

**Prerequisite**: [{{site.prodname}} installed]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/) 
using the standard installation manifests.

Enable application layer policy support for the `calico/node` pod 

- Set `FELIX_POLICYSYNCPATHPREFIX` environment to `"/var/run/nodeagent"` in the `calico-node` DaemonSet.

  ```bash
  kubectl edit daemonset -n kube-system calico-node
  ```

Enable application layer policy in the {{site.prodname}} user interface

- Set `tigera.cnx-manager.alp-support: "true"` in your `tigera-cnx-manager-config` ConfigMap

  ```bash
  kubectl edit configmap -n kube-system tigera-cnx-manager-config
  ## Restart the manager
  kubectl delete pod -n kube-system -l "k8s-app=cnx-manager"
  ```

## Installing Istio

Application layer policy [requires Istio](../requirements#application-layer-policy-requirements).

Install Istio according to the [Istio project documentation](https://istio.io/docs/setup/kubernetes/), making sure to enable mutual TLS authentication. For example:

```bash
curl -L https://git.io/getLatestIstio | sh -
cd $(ls -d istio-*)
kubectl apply -f install/kubernetes/istio-demo-auth.yaml
```

> **Note**: If an "unable to recognize" error occurs after applying `install/kubernetes/istio-demo-auth.yaml` it is likely a race
> condition between creating an Istio CRD and then a resource of that type. Re-run the `kubectl apply`.
{: .alert .alert-info}

## Updating the Istio sidecar injector

The sidecar injector automatically modifies pods as they are created to work
with Istio. This step modifies the injector configuration to add Dikastes, a
{{site.prodname}} component, as sidecar containers.

1. Follow the [Automatic sidecar injection instructions](https://istio.io/docs/setup/kubernetes/sidecar-injection/#automatic-sidecar-injection)
   to install the sidecar injector and enable it in your chosen namespace(s).

1. Apply the following ConfigMap to enable injection of Dikastes alongside Envoy.

   ```bash
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/app-layer-policy/istio-inject-configmap.yaml
   ```

	 > **Note**: You can also
   > [view the manifest in your browser]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/app-layer-policy/istio-inject-configmap.yaml){:target="_blank"}.
   {: .alert .alert-info}

If you would like to install a different version of Istio or inspect the changes
we have made to the standard sidecar injector `ConfigMap`, see
[Customizing the manifests](config-options).

## Adding {{site.prodname}} authorization services to the mesh

Apply the following manifest to configure Istio to query {{site.prodname}} for application layer policy authorization decisions

```bash
kubectl apply -f \
{{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/app-layer-policy/istio-app-layer-policy.yaml
```

> **Note**: You can also
> [view the manifest in your browser](manifests/app-layer-policy/istio-app-layer-policy.yaml){:target="_blank"}.
{: .alert .alert-info}

## Adding namespace labels

Application layer policy is only enforced on pods that are started with the
Envoy and Dikastes sidecars.  Pods that do not have these sidecars will
only enforce standard {{site.prodname}} network policy.

You can control this on a per-namespace basis.  To enable Istio and application
layer policy in a namespace, add the label `istio-injection=enabled`.

	kubectl label namespace <your namespace name> istio-injection=enabled

If the namespace already has pods in it, you will have to recreate them for this
to take effect.

**Note**: Envoy must be able to communicate with the
`istio-pilot.istio-system` service. If you apply any egress policies to your
pods, you *must* enable access. For example, you could
[apply a network policy]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/app-layer-policy/allow-istio-pilot.yaml).
{: .alert .alert-info}
