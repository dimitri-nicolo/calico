## Node requirements

- x86-64 processor with at least 2 cores, 8.0GB RAM and 20 GB free disk space

- Linux kernel 3.10 or later with [required dependencies](#kernel-dependencies).
  The following distributions have the required kernel, its dependencies, and are
  known to work well with {{site.prodname}} and {{include.orch}}.{% if include.orch == "Kubernetes" or include.orch == "host protection" %}
  - CentOS 7
  - Ubuntu 18.04 and 20.04
  - RHEL 7
  - Debian 9
  {% endif %}{% if include.orch == "OpenShift" %}
  - Red Hat Enterprise Linux CoreOS
  {% endif %}{% if include.orch == "OpenStack" %}
  - Ubuntu 18.04
  - CentOS 8
  {% endif %}

- {{site.prodname}} must be able to manage `cali*` interfaces on the host. When IPIP is
  enabled (the default), {{site.prodname}} also needs to be able to manage `tunl*` interfaces.
  When VXLAN is enabled, {{site.prodname}} also needs to be able to manage the `vxlan.calico` interface.

  > **Note**: Many Linux distributions, such as most of the above, include NetworkManager.
  > By default, NetworkManager does not allow {{site.prodname}} to manage interfaces.
  > If your nodes have NetworkManager, complete the steps in
  > [Preventing NetworkManager from controlling {{site.prodname}} interfaces]({{ site.baseurl }}/maintenance/troubleshoot/troubleshooting#configure-networkmanager)
  > before installing {{site.prodname}}.
  {: .alert .alert-info}

- In order to properly run Elasticsearch, nodes must be configured according to the
  [Elasticsearch system configuration documentation.](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html){:target="_blank"}

- The Typha autoscaler requires a minimum number of Linux worker nodes based on total number of schedulable nodes.

  | Total schedulable nodes | Required Linux nodes for Typha replicas |
  |-------------------------|-----------------------------------------|
  | 1                       | 1
  | 2                       | 2
  | 3                       | 3
  | up to 250               | 4
  | up to 500               | 5
  | up to 1000              | 6
  | up to 1500              | 7
  | up to 2000              | 8
  | 2000 or more            | 10

## Key/value store

{{site.prodname}} {{page.version}} requires a key/value store accessible by all
{{site.prodname}} components.
{%- if include.orch == "OpenShift" %}
With OpenShift, the Kubernetes API datastore is used for the key/value store.{% endif -%}
{%- if include.orch == "Kubernetes" %}
On Kubernetes, you can configure {{site.prodname}} to access an etcdv3 cluster directly or to
use the Kubernetes API datastore.{% endif -%}
{%- if include.orch == "OpenStack" %}
For production you will likely want multiple
nodes for greater performance and reliability.  If you don't already have an
etcdv3 cluster to connect to, please refer to {% include open-new-window.html text='the upstream etcd
docs' url='https://coreos.com/etcd/' %} for detailed advice and setup.{% endif %}{% if include.orch == "host protection" %}{% endif %}

## Network requirements

Ensure that your hosts and firewalls allow the necessary traffic based on your configuration.

| Configuration                                                | Host(s)              | Connection type | Port/protocol |
|--------------------------------------------------------------|----------------------|-----------------|---------------|
| {{site.prodname}} networking (BGP)                           | All                  | Bidirectional   | TCP 179 |
| {{site.prodname}} networking in IP-in-IP mode (default mode) | All                  | Bidirectional   | IP-in-IP, often represented by its protocol number `4` |
{%- if include.orch == "OpenShift" %}
| {{site.prodname}} networking with VXLAN enabled              | All                  | Bidirectional   | UDP 4789 |
| Typha access                                                 | Typha agent hosts    | Incoming        | TCP 5473 (default) |
| All                                                         | kube-apiserver host  | Incoming        | Often TCP 443 or 8443\* |
| etcd datastore                                               | etcd hosts           | Incoming        | [Officially](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) TCP 2379 but can vary |
{%- else if include.orch == "Kubernetes" %}
| {{site.prodname}} networking with VXLAN enabled              | All                  | Bidirectional   | UDP 4789 |
| {{site.prodname}} networking with Typha enabled              | Typha agent hosts    | Incoming        | TCP 5473 (default) |
| All                                                          | kube-apiserver host  | Incoming        | Often TCP 443 or 6443\* |
| etcd datastore                                               | etcd hosts           | Incoming        | {% include open-new-window.html text='Officially' url='http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt' %}  TCP 2379 but can vary |
{%- else %}
| All                                                          | etcd hosts           | Incoming        | {% include open-new-window.html text='Officially' url='http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt' %}  TCP 2379 but can vary |
{%- endif %}
{%- if include.orch == "Kubernetes" or include.orch == "OpenShift" %}
| All                                                          | {{site.prodname}} API server hosts | Incoming | TCP 8080 and 5443 (default)
| All                                                          | agent hosts         | Incoming        | TCP 9081 (default)
| All                                                          | Prometheus hosts    | Incoming        | TCP 9090 (default)
| All                                                          | Alertmanager hosts  | Incoming        | TCP 9093 (default)
| All                                                          | ECK operator hosts  | Incoming        | TCP 9443 (default)
| All                                                          | Elasticsearch hosts | Incoming        | TCP 9200 (default)
| All                                                          | Kibana hosts        | Incoming        | TCP 5601 (default)
| All                                                          | {{site.prodname}} Manager host | Incoming | TCP 9443 (default)
| All                                                          | {{site.prodname}} Compliance server host | Incoming | TCP 5443 (default)
{%- endif %}
{%- if include.orch == "Kubernetes" or include.orch == "OpenShift" %}

\* _The value passed to kube-apiserver using the `--secure-port` flag. If you cannot locate this, check the `targetPort` value returned by `kubectl get svc kubernetes -o yaml`._
{% endif -%}
{%- if include.orch == "OpenStack" %}

\* _If your compute hosts connect directly and don't use IP-in-IP, you don't need to allow IP-in-IP traffic._
{%- endif %}

## Privileges

Ensure that {{site.prodname}} has the `CAP_SYS_ADMIN` privilege.

The simplest way to provide the necessary privilege is to run {{site.prodname}} as root or in a privileged container.

{%- if include.orch == "Kubernetes" %}
When installed as a Kubernetes daemon set, {{site.prodname}} meets this requirement by running as a
privileged container. This requires that the kubelet be allowed to run privileged
containers. There are two ways this can be achieved.

- Specify `--allow-privileged` on the kubelet (deprecated).
- Use a {% include open-new-window.html text='pod security policy' url='https://kubernetes.io/docs/concepts/policy/pod-security-policy/' %}.
{% endif -%}
