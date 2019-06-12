
## About enabling application layer policy

Application layer policy for {{site.prodname}} allows you to write policies that
enforce against application layer attributes like HTTP methods or paths as well as
against cryptographically secure identities.

Support for application layer policy is not enabled by default in
{{site.prodname}} installs, since it requires extra CPU and memory resources to
operate.

## Enabling application layer policy

Application layer policy can be enabled during installation of {{ site.prodname }}, or can be enabled on an cluster
already running {{ site.prodname }} 

#### Enabling during installation of {{ site.prodname }}

Prior to applying the `calico.yaml` manifest that will install Calico, modify the file to set the 
`felix-policy-sync-path-prefix` value in the `calico-config` ConfigMap as follows:

```yaml
  felix-policy-sync-path-prefix: "/var/run/nodeagent"
```

Prior to applying the `cnx.yaml` manifest that will install the {{ site.prodname }} Manager and API Server, modify the 
file to set the `tigera.cnx-manager.alp-support` value in the `tigera-cnx-manager-config` ConfigMap as follows:

```yaml
  tigera.cnx-manager.alp-support: "true"
```

#### Enabling after installation of {{ site.prodname }}

For a cluster that is already running {{ site.prodname }}, run the following commands to modify the deployment
to support application layer policy.

```bash
# Update the calico-config ConfigMap to enable application layer policy.
kubectl patch configmap calico-config -n kube-system -p '{"data":{"felix-policy-sync-path-prefix":"/var/run/nodeagent"}}'

# Update the DaemonSet to trigger a rolling upgrade of the calico/node containers (we apply
# an arbitrary label update).
kubectl patch daemonset calico-node -n kube-system -p '{"spec":{"template":{"metadata":{"labels":{"projectcalico.org/application-layer-support":"true"}}}}}'

# Update the tigera-cnx-manager-config ConfigMap to enabled application layer policy support in the UI.
kubectl patch configmap tigera-cnx-manager-config -n kube-system -p '{"data":{"tigera.cnx-manager.alp-support":"true"}}'

# Restart the CNX manager pods to pick up the new config
kubectl delete pod -n kube-system -l "k8s-app=cnx-manager"
```

## Installing Istio

Application layer policy [requires Istio](../requirements#application-layer-policy-requirements).

Install Istio according to the [Istio project documentation](https://istio.io/docs/setup/kubernetes/), making sure to enable mutual TLS authentication. For example:

```bash
curl -L https://git.io/getLatestIstio | ISTIO_VERSION=1.0.7 sh -
cd $(ls -d istio-*)
kubectl apply -f install/kubernetes/helm/istio/templates/crds.yaml
kubectl apply -f install/kubernetes/istio-demo-auth.yaml
```

> **Note**: If an "unable to recognize" error occurs after applying `install/kubernetes/istio-demo-auth.yaml` it is likely a race
> condition between creating an Istio CRD and then a resource of that type. Re-run the `kubectl apply`.
{: .alert .alert-info}

## Updating the Istio sidecar injector

The sidecar injector automatically modifies pods as they are created to work
with Istio. This step modifies the injector configuration to add Dikastes, a
{{site.prodname}} component, as sidecar containers.

1. Follow the [Automatic sidecar injection instructions](https://archive.istio.io/v1.0/docs/setup/kubernetes/sidecar-injection/#automatic-sidecar-injection)
   to install the sidecar injector and enable it in your chosen namespace(s).

1. Apply the following ConfigMap to enable injection of Dikastes alongside Envoy.

   ```bash
   kubectl apply -f \
   {{site.url}}/{{page.version}}/manifests/app-layer-policy/istio-inject-configmap.yaml
   ```

	 > **Note**: You can also
   > [view the manifest in your browser](/{{page.version}}/manifests/app-layer-policy/istio-inject-configmap.yaml){:target="_blank"}.
   {: .alert .alert-info}

If you would like to install a different version of Istio or inspect the changes
we have made to the standard sidecar injector `ConfigMap`, see
[Customizing application layer policy manifests](config-options#customizing-application-layer-policy-manifests).

## Adding {{site.prodname}} authorization services to the mesh

Apply the following manifest to configure Istio to query {{site.prodname}} for application layer policy authorization decisions

```bash
kubectl apply -f \
{{site.url}}/{{page.version}}/manifests/app-layer-policy/istio-app-layer-policy.yaml
```

> **Note**: You can also
> [view the manifest in your browser](/{{page.version}}/manifests/app-layer-policy/istio-app-layer-policy.yaml){:target="_blank"}.
{: .alert .alert-info}

## Adding namespace labels

Application layer policy is only enforced on pods that are started with the
Envoy and Dikastes sidecars.  Pods that do not have these sidecars will
only be protected by standard {{site.prodname}} network policy.

You can control this on a per-namespace basis.  To enable Istio and application
layer policy in a namespace, add the label `istio-injection=enabled`.

```bash
kubectl label namespace <your namespace name> istio-injection=enabled
```

If the namespace already has pods in it, you will have to recreate them for this
to take effect.

> **Note**: Envoy must be able to communicate with the
> `istio-pilot.istio-system` service. If you apply any egress policies to your
> pods, you *must* enable access. For example, you could
> [apply a network policy](/{{page.version}}/manifests/app-layer-policy/allow-istio-pilot.yaml).
>
> If you have no egress policy specified, do not apply the policy example as this will prohibit all egress
> traffic *except* to Istio Pilot.
{: .alert .alert-info}

## Next steps

To get started with application layer policy support, we recommend that you run through the 
[application layer policy tutorial](../tutorials/app-layer-policy).