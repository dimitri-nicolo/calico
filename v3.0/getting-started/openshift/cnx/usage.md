---
title: Using CNX for OpenShift
---

{{site.tseeprodname}} for OpenShift is mostly similar to {{site.tseeprodname}} for Kubernetes, with a few exceptions:

1. {{site.tseeprodname}} settings are tweaked using [the `FelixConfiguration` resource](../../../reference/calicoctl/resources/felixconfig) instead of editing manifests, since {{site.tseeprodname}} is launched as a systemd service instead of a hosted install in OpenShift.
1. A `calicoctl.cfg` file owned by root exists in the default path on each host, which authenticates all {{site.tseeprodname}}
CLI tools (`calicoctl` & `calicoq`) by default without needing to be passed in any etcd connection information, provided they
are run by root user (which is the only user with access to the config file).

More information on these exceptions is covered below.

#### Policy Query with calicoq

Once {{site.tseeprodname}} is installed in OpenShift, each node is automatically configured with
a `calicoctl.cfg` (owned by the root user) which is used by {{site.tseeprodname}} to locate and authenticate
requests to etcd.

To install `calicoq` in OpenShift:

1. Download it to any node.
1. Run it as root user.

See the [calicoq reference](../../../reference/calicoq/) for more information on using `calicoq`.

### Policy Violation Alerting

Policy Violation Alerting is mostly the same in {{site.tseeprodname}} for OpenShift as it is in {{site.tseeprodname}}. Below,
we'll cover how to enable metrics in {{site.tseeprodname}} and how to launch Prometheus using Prometheus-Operator.

#### Enable Metrics

**Prerequisite**: `calicoctl` [installed](../../../usage/calicoctl/install) and [configured](../../../usage/calicoctl/configure).

Enable metrics in {{site.tseeprodname}} for OpenShift by updating the global `FelixConfiguration` resource (`default`) and opening up the necessary port on the host.

{% include {{page.version}}/enable-felix-prometheus-reporting.md %}

1. Allow Prometheus to scrape the metrics by opening up the port on the host:

   ```
   iptables -A INPUT -p tcp --dport 9081 -j ACCEPT
   iptables -I INPUT 1 -p tcp --dport 9081 -j ACCEPT
   ```

#### Configure Prometheus

With metrics enabled, you are ready to monitor `calico/node` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a Prometheus
Operator, Prometheus, and Alertmanager instances for you.

1. Allow Prometheus to run as root:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow Prometheus to configure and use a Security Context.

   ```
   oadm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:prometheus
   ```

1. Allow sleep pod to run with host networking:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Apply the Openshift patches for Prometheus Operator:

   ```
   oc apply -f operator-openshift-patch.yaml
   ```

   >[Click here to view operator-openshift-patch.yaml](operator-openshift-patch.yaml)

1. Apply the Prometheus Operator manifest:

   ```
   oc apply -f operator.yaml
   ```

   >[Click here to view operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml)

1. Apply Prometheus and Alertmanager:

   ```
   oc apply -f monitor-calico.yaml
   ```

   >[Click here to view monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml)

Once running, access Prometheus and Alertmanager using the NodePort from the created service.
See [Policy Violation Alerting](../../../reference/cnx/policy-violations) for more information.

### Policy Audit Mode

See [Policy Auditing](../../../reference/cnx/policy-auditing) for more information.
