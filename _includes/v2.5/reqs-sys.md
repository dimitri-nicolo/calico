## Node requirements

- AMD64 processor with at least 2 cores, 7.5GB RAM and 20 GB free disk space

- Linux kernel 3.10 or later with [required dependencies](#kernel-dependencies).
  The following distributions have the required kernel, its dependencies, and are
  known to work well with {{site.tseeprodname}} and {{include.orch}}.{% if include.orch == "Kubernetes" or include.orch == "host protection" %}
  - CentOS 7
  - Ubuntu 16.04
  - Debian 9
  {% endif %}{% if include.orch == "OpenShift" %}
  - CentOS 7
  {% endif %}{% if include.orch == "OpenStack" %}
  - Ubuntu 16.04
  - CentOS 7
  {% endif %}<br><br>

- {{site.tseeprodname}} must be able to manage `cali*` interfaces on the host. When IPIP is
  enabled (the default), {{site.tseeprodname}} also needs to be able to manage `tunl*` interfaces.

  > **Note**: Many Linux distributions, such as most of the above, include NetworkManager.
  > By default, NetworkManager does not allow {{site.tseeprodname}} to manage interfaces.
  > If your nodes have NetworkManager, complete the steps in
  > [Preventing NetworkManager from controlling {{site.tseeprodname}} interfaces]({{site.baseurl}}/{{page.version}}/maintenance/troubleshooting#configure-networkmanager)
  > before installing {{site.tseeprodname}}.
  {: .alert .alert-info}

- In order to properly run Elasticsearch, nodes must be configured according to the
  [Elasticsearch system configuration documentation.](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html){:target="_blank"}

## Key/value store

{{site.tseeprodname}} {{page.version}} requires a key/value store accessible by all
{{site.tseeprodname}} components. {% if include.orch == "Kubernetes" %} On Kubernetes,
you can configure {{site.tseeprodname}} to access an etcdv3 cluster directly or to
use the Kubernetes API datastore.{% endif %}{% if include.orch == "OpenShift" %} On
OpenShift, {{site.tseeprodname}} can share an etcdv3 cluster with OpenShift, or
you can set up an etcdv3 cluster dedicated to {{site.tseeprodname}}.{% endif %}

{% if include.orch == "OpenStack" %}For production you will likely want multiple
nodes for greater performance and reliability.  If you don't already have an
etcdv3 cluster to connect to, please refer to [the upstream etcd
docs](https://coreos.com/etcd/) for detailed advice and setup.{% endif %}{% if include.orch == "host protection" %}The key/value store must be etcdv3.{% endif %}

## Network requirements

Ensure that your hosts and firewalls allow the necessary traffic based on your configuration.

| Configuration                                                | Host(s)              | Connection type | Port/protocol
|--------------------------------------------------------------|----------------------|-----------------|---------------
| {{site.tseeprodname}} networking (BGP)                           | All                  | Bidirectional   | TCP 179
| {{site.tseeprodname}} networking in IP-in-IP mode (default mode) | All                  | Bidirectional   | IP-in-IP, often represented by its protocol number `4`
{%- if include.orch == "Kubernetes" %}
| {{site.tseeprodname}} networking with Typha enabled              | Typha agent hosts    | Incoming        | TCP 5473 (default)
| All                                                          | kube-apiserver host  | Incoming        | Often TCP 443 or 6443\*
| etcd datastore                                               | etcd hosts           | Incoming        | [Officially](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) TCP 2379 but can vary
{%- else %}
| All                                                          | etcd hosts           | Incoming        | [Officially](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) TCP 2379 but can vary |
{%- endif %}
{%- if include.orch == "OpenShift" %}
| All                                                          | kube-apiserver host  | Incoming        | Often TCP 443 or 8443\*
{%- endif %}
{%- if include.orch == "Kubernetes" or include.orch == "OpenShift" %}
| All                                                          | {{site.tseeprodname}} API server hosts | Incoming | TCP 8080 and 5443 (default)
| All                                                          | agent hosts         | Incoming        | TCP 9081 (default)
| All                                                          | Prometheus hosts    | Incoming        | TCP 9090 (default)
| All                                                          | Alertmanager hosts  | Incoming        | TCP 9093 (default)
| All                                                          | {{site.tseeprodname}} Manager host | Incoming | TCP 30003 and 9443 (defaults)
{%- endif %}
{%- if include.orch != "Kubernetes" %}
| All                                                          | etcd hosts           | Incoming        | [Officially](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) TCP 2379 but can vary

{% endif %}

{%- if include.orch == "Kubernetes" or include.orch == "OpenShift" %}

\* _The value passed to kube-apiserver using the `--secure-port` flag. If you cannot locate this, check the `targetPort` value returned by `kubectl get svc kubernetes -o yaml`._
{% endif -%}
{%- if include.orch == "OpenStack" %}

\* _If your compute hosts connect directly and don't use IP-in-IP, you don't need to allow IP-in-IP traffic._
{% endif -%}

## Privileges

Ensure that {{site.tseeprodname}} has the `CAP_SYS_ADMIN` privilege.

The simplest way to provide the necessary privilege is to run {{site.tseeprodname}} as root or in a privileged container.

{%- if include.orch == "Kubernetes" %}
When installed as a Kubernetes daemon set, {{site.tseeprodname}} meets this requirement by running as a
privileged container. This requires that the kubelet be allowed to run privileged
containers. There are two ways this can be achieved.

- Specify `--allow-privileged` on the kubelet (deprecated).
- Use a [pod security policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/).
{% endif -%}
