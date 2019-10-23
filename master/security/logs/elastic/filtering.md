---
title: Filtering out flow logs
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/filtering
---

{{site.prodname}} supports filtering out flow logs based on user provided
configuration.  Use filtering to suppress logs of low significance.


## Configure flow filtering

Configure filtering based on how {{site.prodname}} was deployed:
- [Tigera operator deployment](#tigera-operator-deployment)
- [Manual deployment](#manual-deployment)

### Tigera operator deployment

Flow log filtering is configured through a ConfigMap in the `tigera-operator`
namespace.

To enable flow log filtering, follow these steps:

1. Create a `filters` directory with a file calld `flow` with the contents of
   your [desired filter](#filter-configuration-files).
   If you are also adding [dns filters](filtering-dns) also add the `dns` file
   to the directory.
1. Create the `fluentd-filters` ConfigMap in the `tigera-operator` namespace
   with the following command.
   ```bash
   kubectl create configmap fluentd-filters -n tigera-operator --from-file=filters
   ```

### Manual deployment

Flow log filtering is configured in the `tigera-es-config` ConfigMap.

To enable flow log filtering, follow these steps:

1. Set the `tigera.elasticsearch.flow-filtering` field in the `tigera-es-config`
   ConfigMap in the `calico-monitoring` Namespace to "true".  This ConfigMap can
   be found in `monitor-calico.yaml`.

1. Set the filters you wish to use in the `tigera.elasticsearch.flow-filters.conf`
   field.  See the following section for more information on writing those filters.

1. Force a rolling update of fluentd by patching the DaemonSet.
   ```bash
   kubectl patch daemonset -n calico-monitoring tigera-fluentd-node -p \
     "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"update-date\":\"`date +'%s'`\"}}}}}"
   ```

## Filter configuration files

The filters defined by the ConfigMap are inserted into the fluentd configuration file.
The [upstream fluentd documentation](https://docs.fluentd.org/filter/grep)
describes how to write fluentd filters.  The [flow log schema](flow) can be referred to
for the specification of the various fields you can filter based on.  Remember to ensure
that the config file is properly indented in the ConfigMap.

### Example 1: filter out a specific namespace

This example filters out all flows whose source or destination namespace is "dev".
Additional namespaces could be filtered by adjusting the regular expression "pattern"s,
or by adding additional `exclude` blocks.

```
<filter flows>
  @type grep
  <exclude>
    key source_namespace
    pattern dev
  </exclude>
  <exclude>
    key dest_namespace
    pattern dev
  </exclude>
</filter>
```

### Example 2: filter out internet traffic to a specific deployment

This example will filter inbound internet traffic to the deployment with pods
named "nginx-internet-*".  Note the use of the `and` directive to only filter
out traffic that is both to the deployment, and from the internet (source `pub`).

```
<filter flows>
  @type grep
  <and>
    <exclude>
        key dest_name_aggr
        pattern ^nginx-internet
    </exclude>
    <exclude>
        key source_name_aggr
        pattern pub
    </exclude>
  </and>
</filter>
```
