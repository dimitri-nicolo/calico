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

- [etcd datastore](/{{page.version}}/usage/{{include.cli}}/configure/etcd)
- [Kubernetes API datastore](/{{page.version}}/usage/{{include.cli}}/configure/kdd)


> **Note**: When running `{{ include.cli }}` inside a container, any environment variables and 
> configuration files must be passed to the container so they are available to 
> the process inside. It can be useful to keep a running container (that sleeps) configured 
> for your datastore, then it is possible to `exec` into the container and have an 
> already configured environment.
{: .alert .alert-info}