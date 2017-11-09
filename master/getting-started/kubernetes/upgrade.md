---
title: Upgrading Calico for Kubernetes
---

This document covers upgrading the Calico components in a Kubernetes deployment.
It also covers upgrading to Tigera CNX from [open source Calico](#upgrading-an-open-source-cluster-with-tigera-cnx),
or from [Tigera Essentials Toolkit](#upgrading-a-cluster-with-tigera-essentials-toolkit-to-cnx).
The upgrade procedure is supported for Calico v1.6+.

It is possible to upgrade the Calico components on a single node without affecting connectivity or
network policy for any existing pods.  However, it is recommended that you do not deploy
new pods to a node that is being upgraded.

It is recommended to upgrade one node at a time, rendering each node as
unscheduleable using [kubectl cordon](http://kubernetes.io/docs/user-guide/kubectl/v1.6/#cordon)
before upgrading the node, and then make the node scheduleable after the upgrade is
complete using [kubectl uncordon](http://kubernetes.io/docs/user-guide/kubectl/v1.6/#uncordon).

> **Note**: When upgrading to etcd v3, as long as the cluster is migrated with the
`etcdctl migrate` command, the v2 data will remain untouched and the etcd v3
server will continue to speak the v2 protocol so the upgrade should have no
impact on Calico.
{: .alert .alert-info}

> **Note**: When upgrading Calico using the Kubernetes datastore driver from a version < v2.3.0
> to a version >= v2.3.0, or when upgrading Calico using the etcd datastore from a version < v2.4.0
> to a version >= v2.4.0, you should follow the steps for
> [upgrading to v1 NetworkPolicy semantics](#upgrading-to-v1-networkpolicy-semantics)
{: .alert .alert-info}

> **Important**: If you are using the Kubernetes datastore and upgrading from
> Calico v2.4.x or earlier to Calico v2.5.x or later, you must
> [migrate your Calico configuration data](https://github.com/projectcalico/calico/blob/master/upgrade/v2.5/README.md)
> before upgrading. Otherwise, your cluster may lose connectivity after the upgrade.
{: .alert .alert-danger}

## Upgrading an Open Source Cluster with Tigera CNX

Follow the steps for [upgrading a hosted installation of Calico](#upgrading-a-hosted-installation-of-calico),
but also [upgrade the additional CNX manifests](#adding-tigera-cnx) after upgrading the core Calico manifests.

## Upgrading a Cluster with Tigera Essentials Toolkit to CNX

Follow the steps for [upgrading a hosted installation of Calico](#upgrading-a-hosted-installation-of-calico),
but also [upgrade the additional CNX manifests](#adding-tigera-cnx) after upgrading the core Calico manifests.  Steps 1 to 3 are not required.

## Adding Tigera CNX
This section covers taking an existing Kubernetes system with Calico and maybe Tigera Essentials Toolkit, and adding Tigera CNX.

#### Prerequisites
This procedure assumes the following:

1. Your system is running the latest 3.0.x release of Calico.  If not, follow the instructions below to upgrade it to the latest 3.0.x release
2. You have obtained the Tigera CNX specific binaries by following the instructions in [getting started]({{site.baseurl}}/{{page.version}}/getting-started/essentials) and uploaded them to a private registry.
3. You have the Calico manifest that was used to install your system available.  This is the manifest which includes the `calico/node` DaemonSet.

#### Prepare for the Upgrade
 Edit your calico manifest:
   - change the calico/node `image:` key to point at the Essentials `calico/node` image in your private registry.
   - add the following to the `env:` section of the `calico/node` Daemonset:
     ```
     - name: FELIX_PROMETHEUSREPORTERENABLED
       value: "true"
     - name: FELIX_PROMETHEUSREPORTERPORT
       value: "9081"
     ```

#### Perform the upgrade
 1. Apply the calico manifest you prepared above with a command like: `kubectl apply -f calico.yaml`
 2. Upgrade each node. Perform the following steps on each node one at a time:
    - First make the node unschedulable:
        ```
        kubectl cordon node-01
        ```
    - Delete the calico-node pod running on the cordoned node and wait for the DaemonSet controller to deploy a replacement using the CNX image.
        ```
        kubectl delete pod -n kube-system calico-node-ajzy6e3t
        ```
    - Once the new calico-node Pod has started, make the node schedulable again.
        ```
        kubectl uncordon node-01
        ```
 3. Install the policy query and violation alerting tools.  For more information about the following instructions, see [Tigera CNX Hosted Install](installation/hosted/essentials/).

    - Configure calico-monitoring namespace and deploy Prometheus Operator by
      applying the [operator.yaml](installation/hosted/essentials/1.6/operator.yaml) manifest.

    ```
    kubectl apply -f operator.yaml
    ```

    - Wait for third party resources to be created. Check by running:

    ```
    $ kubectl get thirdpartyresources --watch
    ```

    - Apply the [monitor-calico.yaml](installation/hosted/essentials/1.6/monitor-calico.yaml) manifest which will
      install prometheus and alertmanager.

    ```
    $ kubectl apply -f monitor-calico.yaml
    ```

4. Add Tigera CNX Manager.  Note that this step may require API downtime, 
   because the API server's command line flags will probably need changing.
   For more information about the following instructions, see [Tigera CNX Hosted Install](installation/hosted/essentials/).

   - [Decide on an authentication method, and configure Kubernetes]({{site.baseurl}}/{{page.version}}/reference/essentials/authentication).

   - Edit and apply [the manifest](installagion/hosted/essentials/1.6/cnx-manager.yaml) defining the CNX Manager resources.
     The `tigera-cnx-manager-web-config` ConfigMap at the start of the file
     defines two parameters that may need changing: the OIDC client id
     (only if using Google login), and the Kubernetes API location (must
     be reachable from any system running the web application).

     ```
     # Edit the ConfigMap first
     kubectl apply -f cnx-manager.yaml
     ```

   - Define RBAC permissions for users to access CNX Manager.
     [This document]({{site.baseurl}}/{{page.version}}/reference/essentials/rbac-tiered-policies) describes how to do that.

## Upgrading a Hosted Installation of Calico

This section covers upgrading a [self-hosted]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted) Calico installation.

Note that while a self-hosted installation of Calico is typically done all at once (via calico.yaml), it is
recommended to perform upgrades one component at a time.

#### Upgrading the Calico Kubernetes controllers

In a self-hosted Calico installation, the calico/kube-controllers container is run as a deployment.  As such,
it can be upgraded via the standard [deployment mechanism](http://kubernetes.io/docs/user-guide/deployments/#updating-a-deployment).

To upgrade the controllers, simply apply changes to the deployment specification and Kubernetes will
do the rest.

```
kubectl apply -f new-controllers.yaml
```

> **Note**: The deployment must use `.spec.strategy.type==Recreate` to
> ensure that at most one instance of the controller is running at a time.
{: .alert .alert-info}


#### Upgrading the Calico DaemonSet

Upgrading the CNI plugin or calico/node image is done through a DaemonSet.  DaemonSets do not
currently support an update operation, and as such must be updated manually.

To upgrade the DaemonSet:

##### 1. Apply changes to the existing DaemonSet via kubectl apply.

Modify the DaemonSet manifest and run:

```
kubectl apply -f calico-node.yaml
```

> **Note**: Alternatively, you can use `kubectl edit` to modify the DaemonSet.
{: .alert .alert-info}


##### 2. Upgrade each node.

Perform the following steps on each node one at a time.

First make the node unschedulable:

```
kubectl cordon node-01
```

Delete the calico-node pod running on the cordoned node and wait for the
DaemonSet controller to deploy a replacement.

```
kubectl delete pod -n kube-system calico-node-ajzy6e3t
```

Once the new calico-node Pod has started, make the node schedulable again.

```
kubectl uncordon node-01
```


> **Note**: You may want to pre-fetch the new Docker image to ensure the new
> node image is started within BIRD's graceful restart period of 90 seconds.
{: .alert .alert-info}


#### Updating the Calico ConfigMap

Most self-hosted Calico deployments use a ConfigMap for configuration of the Calico
components.

To update the ConfigMap, make any desired changes and apply the new ConfigMap using
kubectl.  You will need to restart the Calico Kubernetes controllers and each calico/node instance
as described above before new config is reflected.

## Upgrading Components Individually

This section covers upgrading each component individually for use with custom configuration
management tools.

#### Upgrading the calico/node container

The calico/node container runs on each node in a Kubernetes cluster.  It runs Felix for policy
enforcement and BIRD for BGP networking (when enabled).

To upgrade the calico/node container:

- Pull the new version of the calico/node image to each node.  e.g `docker pull quay.io/calico/node:vA.B.C`
- Update the image in your process management to reference the new version.
- Stop the running calico/node container, and start it with the newly pulled version.

#### Upgrading the Calico CNI plugins

The Calico CNI plugins (calico and calico-ipam) are typically installed in /opt/cni/bin, though
this can vary based on deployment.

To upgrade the plugins, simply remove the existing binaries and replace them with the desired version.

To upgrade the CNI config (typically located in /etc/cni/net.d) simply make the desired changes to the
config file.  It will be picked up by the kubelet automatically for Kubernetes v1.4.0+.  For older versions
of Kubernetes you must restart the kubelet for changes to be applied.

#### Upgrading the Calico Kubernetes controllers

The calico/kube-controllers pod can be stopped and restarted without affecting connectivity or
policy on existing pods.  New pods in existing Namespaces will correctly have
existing policy applied even when the controller is not running.  However, when the
controllers are not running:

- New NetworkPolicies will not be applied.
- New Pods in new Namespaces will not get network connectivity.
- Label changes to existing pods will not be reflected in the applied policy.


> **Note**: Only one instance of the controller should ever be active at a time.
{: .alert .alert-info}

To upgrade the controllers:

- Pull the new version of the calico/kube-controllers image to each node.
- Update the image in your process management to reference the new version.
- Stop the running container, and start it with the newly pulled version.

We recommend running the controllers as a Kubernetes deployment with type "recreate", in which
case upgrade can be handled entirely through the
standard [deployment mechanism](http://kubernetes.io/docs/user-guide/deployments/#updating-a-deployment)

## Upgrading to v1 NetworkPolicy semantics

Calico v2.3.0 (when using the Kubernetes datastore driver) and Calico v2.4.0 (when using the etcd datastore driver)
interpret the Kubernetes `NetworkPolicy` differently than previous releases, as specified
in [upstream Kubernetes](https://github.com/kubernetes/kubernetes/pull/39164#issue-197243974).

To maintain behavior when upgrading, you should follow these steps prior to upgrading Calico to ensure your configured policy is
enforced consistently throughout the upgrade process.

- In any Namespace that previously did _not_ have a "DefaultDeny" annotation:
  - Delete any NetworkPolicy objects in that Namespace.  After upgrade, these policies will become active and may block traffic that was previously allowed.
- In any Namespace that previously had a "DefaultDeny" annotation:
  - Create a NetworkPolicy which matches all pods but does not allow any traffic.  After upgrade, the Namespace annotation will have no effect, but this empty NetworkPolicy will provide the same behavior.

Here is an example of a NetworkPolicy which selects all pods in the Namespace, but does not allow any traffic:

```yaml
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: default-deny
spec:
  podSelector:
```

> **Note**: The above steps should be followed when upgrading to
> Calico v2.3.0+ using the Kubernetes
> datastore driver, and Calico v2.4.0+ using the etcd datastore,
> independent of the Kubernetes version being used.
{: .alert .alert-info}
