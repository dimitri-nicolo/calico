---
title: CNX for Kubernetes Demo
---

This guide is a variation of the simple policy demo intended to introduce the extra features of {{site.prodname}} to people already familiar with Project Calico for Kubernetes.

It requires a Kubernetes cluster configured with Calico networking and {{site.prodname}}, and expects that you have `kubectl` configured to interact with the cluster.

You can quickly and easily obtain such a cluster by following one of the
[installation guides]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation),
or by [upgrading an existing cluster]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/upgrade).

The key steps in moving to {{site.prodname}} are to change to the {{site.prodname}} version of calico-node, update its configuration, download calicoq and deploy Prometheus.

This guide assumes that you have applied all the example manifests in the [examples directory]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/1.6/)
and that your cluster consists of the following nodes:
  * k8s-node1
  * k8s-node2
  * k8s-master

Where you see references to these in the text below, substitute for your actual node names.

### Configure Namespaces

This guide will deploy pods in a Kubernetes Namespaces.  Let's create the `Namespace` object for this guide.

```
kubectl create ns policy-demo
```

### Create demo Pods

We'll use Kubernetes `Deployment` objects to easily create pods in the `Namespace`.

1) Create some nginx pods in the `policy-demo` Namespace, and expose them through a Service.

```shell
# Run the Pods.
kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx

# Create the Service.
kubectl expose --namespace=policy-demo deployment nginx --port=80
```

2) Ensure the nginx service is accessible.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q nginx -O -
```

You should see a response from `nginx`.  Great! Our Service is accessible.  You can exit the Pod now.

Now let's inspect the network policies using calicoq.  calicoq complements calicoctl by inspecting the
dynamic aspects of {{site.prodname}} Policy: in particular displaying the endpoints actually affected by policies,
and the policies that actually apply to endpoints.

The full calicoq documentation is [here]({{site.baseurl}}/{{page.version}}/reference/calicoq).
```
# Point calicoq at etcd / the Kubernetes API Server in the same way as calicoctl.  You can also use a config file.
# The host command displays information about the policies that select endpoints on a host.
ETCD_ENDPOINTS=http://10.96.232.136:6666 ./calicoq host k8s-node1
Policies that match each endpoint:

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  # These are the policies that apply directly to the endpoint.  calicoq can display both
  # {{site.prodname}} Policies and Kubernetes NetworkPolicies, although this example focuses on the latter.
  # They're listed in the order they apply.
  Policies:
    # These first two policies are defined in the calico-monitoring.yaml manifest.
    # The selectors here have been translated from the original NetworkPolicies to the {{site.prodname}}
    # format (note the addition of the namespace test).
    Policy "calico-monitoring.calico-node-alertmanager" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'alertmanager' && alertmanager == 'calico-node-alertmanager'")
    Policy "calico-monitoring.calico-node-alertmanager-mesh" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'alertmanager' && alertmanager == 'calico-node-alertmanager'")
    # This policy and the profile following it are created automatically by the policy controller.
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.calico-monitoring
  # These are the policies that match the endpoint in their rules.
  Matched by policies:
    Policy calico-monitoring.calico-node-alertmanager-mesh (rule 0 inbound source match; selector "app in { "alertmanager" } && alertmanager in { "calico-node-alertmanager" } && calico/k8s_ns == 'calico-monitoring'")

...

Workload endpoint k8s/calico-monitoring.prometheus-calico-node-prometheus-0/eth0
  Policies:
    Policy "calico-monitoring.prometheus" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'prometheus' && prometheus == 'calico-node-prometheus'")
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.calico-monitoring

# This endpoint has no NetworkPolicies configured - just the default Calico policy to allow traffic.
Workload endpoint k8s/policy-demo.nginx-2371676037-7w78m/eth0
  Policies:
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.policy-demo

...
```

### Enable isolation

Let's turn on isolation in our policy-demo Namespace. {{site.prodname}} will then prevent connections to pods in this Namespace.

Running the following command creates a NetworkPolicy which implements a default deny behavior for all pods in the `policy-demo` Namespace.

```
kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: extensions/v1beta1
metadata:
  name: default-deny
  namespace: policy-demo
spec:
  podSelector:
    matchLabels: {}
EOF
```

#### Test Isolation

This will prevent all access to the nginx Service.  We can see the effect by trying to access the Service again.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
wget: download timed out
/ #
```

The request should time out after 5 seconds.  By enabling isolation on the Namespace, we've prevented access to the Service.

### Denied packet metrics and Alerting
Now would be a great time to take a look at the denied packet metrics.  Get the service listing from kubectl:
```
kubectl get svc -n calico-monitoring
NAME                       CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
alertmanager-operated      None             <none>        9093/TCP,6783/TCP   6h
calico-node-alertmanager   10.105.253.248   <nodes>       9093:30903/TCP      6h
calico-node-prometheus     10.105.26.250    <nodes>       9090:30909/TCP      6h
prometheus-operated        None             <none>        9090/TCP            6h

```
This tells us that the `calico-node-prometheus` service is running using a NodePort on port 30909. Point a web browser at [http://k8s-node1:30909/graph](http://k8s-node1:30909/graph).

If you click on the drop down box `- insert metric at cursor -`, you should see a list of metrics which are available:
 - `calico_denied_packets`
 - `calico_denied_bytes`
 - `up`
 - `scrape_duration_seconds`
 - `scrape_samples_post_metric_relabelling`
 - `scrape_samples_scraped`

The first 3 in the list above are useful for monitoring your deployment, while the last 3 are useful for monitoring the health of your monitoring system.

Note that if you have not sent any denied packets recently, `calico_denied_packets` and `calico_denied_bytes` may not appear in the drop down.

Select `calico_denied_packets` and click the `Execute` button.  The `console` tab should now show a key like this:
```calico_denied_packets{endpoint="calico-metrics-port",instance="10.240.0.16:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-zs6gt",policy="profile/k8s_ns.policy-demo/0/deny",service="calico-node-metrics",srcIP="192.168.213.12"}``` and a value.

This indicates that the pod `calico-node-zs6gt` has reported 3 denied packets from `192.168.213.12` and that the packets were denied by the `profile/k8s_ns.policy-demo/0/deny` - which is the namespace default deny rule you enabled above using the namespace annotation.

If you now click on the `graph` tab, you will see a graph of the denied packet count against time:
![Graph Example]({{site.baseurl}}/images/Graph.png)

Prometheus can also do some calculations based on metrics - for example to show the *rate* of denied packets.  Update the expression text box to contain the expression `rate(calico_denied_packets[10s])` and click execute again.  The graph and table will now show you the rate of denied packets averaged over the last 10s.

Now let's try setting off an alert.  If you click on 'Alerts', you'll see that we have a `DeniedPacketsRate` alert configured. Notice that it is green to indicate that it is not currently firing. If you click on it, it will show you that it will fire if `rate(calico_denied_packets[10s]) > 50`.

If you click on that expression, it will skip back to the graph screen and show you all the times that rate of packets was > 50.  So let's send some denied packets.  In the `access` pod you created above, run the following command:
```
for i in `seq 1 10000`; do (wget -q --timeout=1 nginx -O - & sleep 0.01); done
```
Refresh the graph and you should see some data points appear.  Now switch back to the Alerts page and see that the Alert fires.  You may need to refresh the Alerts page a few times until the Alert goes red.  Now if you click on the Alert, it will show you the combination of labels that is firing the Alert:
![Alert Example]({{site.baseurl}}/images/Alert.png)

### Allow Access using a NetworkPolicy

Now, let's enable access to the nginx Service using a NetworkPolicy.  This will allow incoming connections from our `access` Pod, but not
from anywhere else.

Create a network policy `access-nginx` with the following contents:

```
kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: extensions/v1beta1
metadata:
  name: access-nginx
  namespace: policy-demo
spec:
  podSelector:
    matchLabels:
      run: nginx
  ingress:
    - from:
      - podSelector:
          matchLabels:
            run: access
EOF
```

> **Note**: The NetworkPolicy allows traffic from Pods with the label `run: access`
> to Pods with the label `run: nginx`.  These are the labels automatically added to
> Pods started via `kubectl run` based on the name of the `Deployment`.
{: .alert .alert-info}


We should now be able to access the Service from the `access` Pod.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

However, we still cannot access the Service from a Pod without the label `run: access`:

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo cant-access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/cant-access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
wget: download timed out
/ #
```

You can clean up the demo by deleting the demo Namespace:

```shell
kubectl delete ns policy-demo
```

This was just a simple example of the Kubernetes NetworkPolicy API and how {{site.prodname}} can secure your Kubernetes cluster.  For more
information on network policy in Kubernetes, see the [Kubernetes user-guide](http://kubernetes.io/docs/user-guide/networkpolicies/).

For a slightly more detailed demonstration of Policy, check out the [stars demo]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/tutorials/stars-policy/).
