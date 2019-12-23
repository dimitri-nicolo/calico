---
title: CNX for Kubernetes Demo
---

This guide is a variation of the simple policy demo intended to introduce the extra features of {{site.tseeprodname}} to people already familiar with Project Calico for Kubernetes.

It requires a Kubernetes cluster configured with Calico networking and {{site.tseeprodname}}, and expects that you have `kubectl` configured to interact with the cluster.

You can quickly and easily obtain such a cluster by following one of the
[installation guides]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation),
or by [upgrading an existing cluster]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/upgrade-cnx).

The key steps in moving to {{site.tseeprodname}} are to change to the {{site.tseeprodname}} version of calico-node, update its configuration, download calicoq and deploy Prometheus.

This guide assumes that you have installed all the CNX components from the
guides above and that your cluster consists of the following nodes:
  * k8s-node1
  * k8s-node2
  * k8s-master

Where you see references to these in the text below, substitute for your actual node names.

### Configure Namespaces

This guide will deploy pods in a Kubernetes namespace.  Let's create the `Namespace` object for this guide.

```
kubectl create ns policy-demo
```

### Create demo Pods

We'll use Kubernetes `Deployment` objects to easily create pods in that namespace.

1) Create some nginx pods in the `policy-demo` namespace

```shell
kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx
```

and expose them through a service on port 80.

```shell
kubectl expose --namespace=policy-demo deployment nginx --port=80
```

2) Check that the nginx service is accessible, by trying to access it from
another, busybox pod.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Once the pod is running, attempt to access the nginx service.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q nginx -O -
```

You should see a response from `nginx`.  Great! Our service is accessible.  You
can exit the busybox pod now.

3) Inspect the network policies using calicoq.  The `host` command displays
information about the policies for endpoints on a given host.

> **Note**: calicoq complements calicoctl by inspecting the
> dynamic aspects of {{site.tseeprodname}} Policy: in particular displaying the endpoints actually affected by policies,
> and the policies that actually apply to endpoints.
>
> The full calicoq documentation is [here]({{site.baseurl}}/{{page.version}}/reference/calicoq).
{: .alert .alert-info}

```
ETCD_ENDPOINTS=http://10.96.232.136:6666 ./calicoq host k8s-node1
```

You should see the following output.

```
Policies and profiles for each endpoint on host "k8s-node1":

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  Policies:
    Policy "calico-monitoring/knp.default.calico-node-alertmanager" (order 1000; selector "(projectcalico.org/orchestrator == 'k8s' && alertmanager == 'calico-node-alertmanager' && app == 'alertmanager') && projectcalico.org/namespace == 'calico-monitoring'")
    Policy "calico-monitoring/knp.default.calico-node-alertmanager-mesh" (order 1000; selector "(projectcalico.org/orchestrator == 'k8s' && alertmanager == 'calico-node-alertmanager' && app == 'alertmanager') && projectcalico.org/namespace == 'calico-monitoring'")
    Policy "calico-monitoring/knp.default.default-deny" (order 1000; selector "(projectcalico.org/orchestrator == 'k8s') && projectcalico.org/namespace == 'calico-monitoring'")
  Profiles:
    Profile "kns.calico-monitoring"
  Rule matches:
    Policy "calico-monitoring/knp.default.calico-node-alertmanager-mesh" inbound rule 1 source match; selector "(projectcalico.org/namespace == 'calico-monitoring') && (projectcalico.org/orchestrator == 'k8s' && app in { 'alertmanager' } && alertmanager in { 'calico-node-alertmanager' })"

...

Workload endpoint k8s/policy-demo.nginx-8586cf59-5bxvh/eth0
  Policies:
  Profiles:
    Profile "kns.policy-demo"
```

For each workload endpoint, the `Policies:` section lists the policies that
apply to that endpoint, in the order they apply.  calicoq displays both
{{site.tseeprodname}} Policies and Kubernetes NetworkPolicies, although this
example focuses on the latter.  The `Rule matches:` section lists the
policies that match that endpoint in their rules, in other words that have
rules that deny or allow that endpoint as a packet source or destination.

Focusing on the
`k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0` endpoint:

- The first two policies are defined in the calico-monitoring.yaml manifest.
  The selectors here have been translated from the original NetworkPolicies to
  the {{site.tseeprodname}} format (note the addition of the namespace test).

- The third policy and the following profile are created automatically by the
  policy controller.

4) Use calicoctl to see the detail of any particular policy or profile.  For
example, for the `kns.policy-demo` profile, which defines default behavior for
pods in the `policy-demo` namespace:

```
ETCD_ENDPOINTS=http://10.96.232.136:6666 ./calicoctl get profile kns.policy-demo -o yaml
```

You should see the following output.

```
apiVersion: projectcalico.org/v3
kind: Profile
metadata:
  creationTimestamp: 2018-01-09T10:20:52Z
  name: kns.policy-demo
  resourceVersion: "661"
  uid: c541b088-f526-11e7-a837-42010a80000a
spec:
  egress:
  - action: Allow
    destination: {}
    source: {}
  ingress:
  - action: Allow
    destination: {}
    source: {}
```

### Enable isolation

Let's turn on isolation in our policy-demo namespace. {{site.tseeprodname}} will then prevent connections to pods in this namespace.

Running the following command creates a NetworkPolicy which implements a default deny behavior for all pods in the `policy-demo` namespace.

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

> **Note**: Although that NetworkPolicy spec does not explicitly deny or drop
> any packets, it has a 'default deny' effect because [CNX
> semantics]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier#how-policy-is-evaluated)
> are that a packet will be dropped if there are policies applying to an
> endpoint, but those policies take no action on that packet.
>
> This is also why the denied packet metrics below have
> `policy="default/no-policy-match-inbound/0/deny"` and not
> `policy="policy-demo/knp.default.default-deny/0/deny"`.  `no-policy-match`
> represents the CNX semantics as above.
{: .alert .alert-info}

#### Test Isolation

This will prevent all access to the nginx service.  We can see the effect by trying to access the service again.
Start another pod within the `policy-demo` namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Attempt to connect to the nginx service.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

You should see the following output.

```
wget: download timed out
```

The request should time out after 5 seconds.  By enabling isolation on the namespace, we've prevented access to the service.

### Denied packet metrics and Alerting
Now would be a great time to take a look at the denied packet metrics.  Get the service listing from kubectl:

```
kubectl get svc -n calico-monitoring
```

You should see the following output.

```
NAME                       CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
alertmanager-operated      None             <none>        9093/TCP,6783/TCP   6h
calico-node-alertmanager   10.105.253.248   <nodes>       9093:30903/TCP      6h
calico-node-prometheus     10.105.26.250    <nodes>       9090:30909/TCP      6h
prometheus-operated        None             <none>        9090/TCP            6h

```

This tells us that the `calico-node-prometheus` service is running using a NodePort on port 30909. Point a web browser at `http://k8s-node1:30909/graph`.

If you click on the drop down box `- insert metric at cursor -`, you should see a list of metrics which are available:
 - `calico_denied_packets`
 - `calico_denied_bytes`
 - `up`
 - `scrape_duration_seconds`
 - `scrape_samples_post_metric_relabelling`
 - `scrape_samples_scraped`

The first 3 in the list above are useful for monitoring your deployment, while the last 3 are useful for monitoring the health of your monitoring system.

Note that if you have not sent any denied packets recently, `calico_denied_packets` and `calico_denied_bytes` may not appear in the drop down.

Select the `Console` tab, then in the text box at the top of the page type `calico_denied_packets[10m]` and Enter (or click the `Execute` button).  The `Console` tab should now show `calico_denied_packets` metrics for the last 10 minutes:
```
calico_denied_packets{endpoint="calico-metrics-port",instance="10.240.0.16:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-zs6gt",policy="default/no-policy-match-inbound/0/deny",service="calico-node-metrics",srcIP="192.168.213.12"}
```
and a value and timestamp

This indicates that the pod `calico-node-zs6gt` has reported 3 denied packets from `192.168.213.12`.

If you now click on the `Graph` tab and change the expression to just `calico_denied_packets`, you will see a graph of the denied packet count against time:
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

Now, let's enable access to the nginx service using a NetworkPolicy.  This will allow incoming connections from our `access` pod, but not
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

> **Note**: The NetworkPolicy allows traffic from pods with the label `run: access`
> to pods with the label `run: nginx`.  These are the labels automatically added to
> pods started via `kubectl run` based on the name of the `Deployment`.
{: .alert .alert-info}


We should now be able to access the service from the `access` pod.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Attempt to access the service again.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

You should see an HTTP response.

However, we still cannot access the service from a pod without the label `run: access`:

```
kubectl run --namespace=policy-demo cant-access --rm -ti --image busybox /bin/sh
```

Attempt to access the service again.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

After 5 seconds, you should see the following output.

```
wget: download timed out
```

You can clean up the demo by deleting the demo namespace:

```shell
kubectl delete ns policy-demo
```

This was just a simple example of the Kubernetes NetworkPolicy API and how {{site.tseeprodname}} can secure your Kubernetes cluster.  For more
information on network policy in Kubernetes, see the [Kubernetes user guide](http://kubernetes.io/docs/user-guide/networkpolicies/).

For a slightly more detailed demonstration of Policy, check out the [Stars Policy Demo]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/tutorials/stars-policy/).
