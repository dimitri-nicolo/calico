Many `{{ include.cli }}` commands require access to the {{site.prodname}} datastore. In most
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

- [etcd datastore](/{{page.version}}/usage/{{include.cli}}/configure/etcd)
- [Kubernetes API datastore](/{{page.version}}/usage/{{include.cli}}/configure/kdd)

> **Note**: When running `{{ include.cli }}` inside a container, any environment variables and 
> configuration files must be passed to the container so they are available to 
> the process inside. It can be useful to keep a running container (that sleeps) configured 
> for your datastore, then it is possible to `exec` into the container and have an 
> already configured environment.
{: .alert .alert-info}

{% if include.cli == "calicoq" %}
#### {{site.prodname}} Federation

If you are using [{{site.prodname}} Federation](/{{page.version}}/usage/federation/index) and you wish to view the
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

{% endif %}

