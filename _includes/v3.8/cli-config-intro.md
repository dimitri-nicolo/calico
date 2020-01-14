Many `{{ include.cli }}` commands require access to the {{site.tseeprodname}} datastore. In most
circumstances, `{{ include.cli }}` cannot achieve this connection by default. You can provide
`{{ include.cli }}` with the information it needs using either of the following.

1. **Configuration file**: by default, `{{ include.cli }}` will look for a configuration file
at `/etc/calico/calicoctl.cfg`. You can override this using the `--config` option with
commands that require datastore access. The file can be in either YAML or JSON format.
It must be valid and readable by `{{ include.cli }}`. A YAML example follows.

   ```
   apiVersion: projectcalico.org/v3
   kind: CalicoAPIConfig
   metadata:
   spec:
     datastoreType: "etcdv3"
     etcdEndpoints: "http://etcd1:2379,http://etcd2:2379"
     ...
   ```

1. **Environment variables**: If `{{ include.cli }}` cannot locate, read, or access a configuration
file, it will check a specific set of environment variables.

Refer to the section that corresponds to your datastore type for a full set of options
and examples.

- [etcd datastore](/{{page.version}}/getting-started/{{include.cli}}/configure/etcd)
- [Kubernetes API datastore](/{{page.version}}/getting-started/{{include.cli}}/configure/kdd)

> **Note**: When running `{{ include.cli }}` inside a container, any environment variables and
> configuration files must be passed to the container so they are available to
> the process inside. It can be useful to keep a running container (that sleeps) configured
> for your datastore, then it is possible to `exec` into the container and have an
> already configured environment.
{: .alert .alert-info}

{% if include.cli == "calicoq" %}
#### {{site.tseeprodname}} Federation

If you are using [{{site.tseeprodname}} Federation](/{{page.version}}/networking/federation/index) and you wish to view the
remote cluster endpoints using `{{ include.cli }}` then it is also necessary to include any files (kubeconfigs,
certificates and keys) that are referenced in the Remote Cluster Configuration resources in the same location as
specified in these resources. For example, suppose you have a Remote Cluster Configuration resource that references a
kubeconfig file at `/etc/tigera-federation-remotecluster/kubeconfig` then that kubeconfig would need to be at
that same location on the local filesystem where you are running `{{ include.cli }}`. If the file is missing then that
remote cluster configuration will not be included in the `{{ include.cli }}` output, and will instead indicate
an error for that cluster.

> **Note**: When running `{{ include.cli }}` inside a container or as a kubernetes pod these files need to be mounted
> into the container at the correct location within the container.
{: .alert .alert-info}

#### {{site.tseeprodname}} AWS Security Group Integration

If you are using
[{{site.tseeprodname}} AWS Security Group Integration](/{{page.version}}/getting-started/kubernetes/installation/aws-sg-integration)
some additional environment variables need to be provided to `{{include.cli}}`
to ensure endpoints have the proper labels when they are evaluated.

If you are using the
[Kubernetes pod method](/{{page.version}}/getting-started/calicoq/#installing-calicoq-as-a-kubernetes-pod)
for running `{{include.cli}}` these environment variables will be read from a
ConfigMap so there is no additional configuration necessary.

If you are running `{{include.cli}}` as a binary or container on a single host
you will need to ensure the following environment variables are set with the
appropriate values from the `tigera-aws-config` ConfigMap.

| Environment variable              | ConfigMap Key |
| --------------------------------- | ------------- |
| `TIGERA_DEFAULT_SECURITY_GROUPS`  | `default_sgs` |
| `TIGERA_POD_SECURITY_GROUP`       | `pod_sg`      |

> **Note**: Retrieving the values from the ConfigMap must be done after installing
> the AWS Security Group Integration as the ConfigMap will not exist before installation.
{: .alert .alert-info}
{% endif %}

