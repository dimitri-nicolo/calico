---
title: Configuring calicoctl to connect to the Kubernetes API datastore
canonical_url: https://docs.tigera.io/v2.3/usage/calicoctl/configure/kdd
---


{% include {{page.version}}/cli-config-kdd.md %}


## Examples

#### Kubernetes command line

```
DATASTORE_TYPE=kubernetes KUBECONFIG=~/.kube/config calicoctl get nodes
```

#### Example configuration file

```yaml
apiVersion: projectcalico.org/v3
kind: CalicoAPIConfig
metadata:
spec:
  datastoreType: "kubernetes"
  kubeconfig: "/path/to/.kube/config"
```

#### Example using environment variables

```shell
$ export DATASTORE_TYPE=kubernetes
$ export KUBECONFIG=~/.kube/config
$ calicoctl get workloadendpoints
```

And using `CALICO_` prefixed names:

```shell
$ export CALICO_DATASTORE_TYPE=kubernetes
$ export CALICO_KUBECONFIG=~/.kube/config
$ calicoctl get workloadendpoints
```


### Checking the configuration

Here is a simple command to check that the installation and configuration is
correct.

```
calicoctl get nodes
```

A correct setup will yield a list of the nodes that have registered.  If an
empty list is returned you are either pointed at the wrong datastore or no
nodes have registered.  If an error is returned then attempt to correct the
issue then try again.


### Next steps

Now you are ready to read and configure most aspects of {{site.tseeprodname}}.  You can
find the full list of commands in the
[Command Reference]({{site.baseurl}}/{{page.version}}/reference/calicoctl/commands/).

The full list of resources that can be managed, including a description of each,
can be found in the
[Resource Definitions]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/).
