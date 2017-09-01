---
title: Using Essentials for OpenShift
---

Tigera Essentials for OpenShift is mostly similar to Tigera Essentials for Kubernetes, with a few exceptions:

1. Calico settings are tweaked using `calicoctl config set` command instead of editing manifests, since
Calico is launched as a systemd service instead of a hosted install in OpenShift.
1. A more static Prometheus is launched instead of Prometheus Operator since Third Party Resources
are not supported by OpenShift.
1. A `calicoctl.cfg` file owned by root exists in the default path on each host, which authenticates all Calico
CLI tools (`calicoctl` & `calicoq`) by default without needing to be passed in any etcd connection information, provided they
are run by root user (which is the only user with access to the config file).

More information on these exceptions is covered below.

#### Policy Query with calicoq

Once Calico is installed in OpenShift, each node is automatically configured with
a `calicoctl.cfg` (owned by the root user) which is used by Calico to locate and authenticate
requests to etcd. 

To install `calicoq` in OpenShift:

1. Download it to any node.
1. Run it as root user.

See the [calicoq reference](http://localhost:4000/v2.4/reference/calicoq/) for more information on using `calicoq`.

### Policy Violation Alerting

Policy Violation Alerting is mostly the same in Essentials for OpenShift for Calico, but monitoring of the metrics
cannot be done using Prometheus Operator, as Third Party Resources are not supported in OpenShift. Below,
we'll cover how to enable metrics in Calico and how to launch Prometheus without using Prometheus-Operator.

##### Enable Metrics

1. Enable metrics in Essentials for OpenShift by setting a Felix Global Config setting:

   ```
   sudo calicoctl config set --raw=felix PrometheusReporterEnabled true
   sudo calicoctl config set --raw=felix PrometheusReporterPort 9081
   ```

1. Allow Prometheus to scrape the metrics by opening up the port on the host:

   ```
   iptables -A INPUT -p tcp --dport 9081 -j ACCEPT
   iptables -I INPUT 1 -p tcp --dport 9081 -j ACCEPT
   ```

##### Configure Prometheus

With metrics enabled, you are ready to monitor `calico/node` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a basic
instance for you.

1. Allow Prometheus to run as root:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow sleep pod to run with host networking:
   
   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Apply Prometheus:

   ```
   oc apply -f http://{{site.baseurl}}/{{page.version}}/getting-started/openshift/calico-monitoring.yml
   ```

Once running, access Prometheus and Alertmanager using the NodePort from the created service.
See [Policy Violation Alerting]({{site.baseurl}}/{{page.version}}/reference/essentials/policy-violations) for more information.

##### Updating Rules

Because Prometheus-Operator is not being used, updates made to the rules in the `calico-prometheus-dp-rate` ConfigMap
will not get picked up by Prometheus until a SIGHUP signal is explicitly issued.

### Policy Audit Mode

See [Policy Auditing](http://{{site.baseurl}}/{{page.version}}/reference/essentials/policy-auditing) for more information.
