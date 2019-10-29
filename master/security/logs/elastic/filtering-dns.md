---
title: Filtering out DNS logs
canonical_url: https://docs.tigera.io/master/security/logs/elastic/filtering-dns
---

{{site.prodname}} supports filtering out DNS logs based on user provided
configuration.  Use filtering to suppress logs of low significance.

## Configure DNS filtering

Configure filtering based on how {{site.prodname}} was deployed:
- [Operator deployment](#operator-deployment)
- [Manual/Helm deployment](#manualhelm-deployment)

### Operator deployment

DNS log filtering is configured through a ConfigMap in the `tigera-operator`
namespace.

To enable DNS log filtering, follow these steps:

1. Create a `filters` directory with a file calld `dns` with the contents of
   your [desired filter](#filter-configuration-files).
   If you are also adding [flow filters](filtering) also add the `flow` file
   to the directory.
1. Create the `fluentd-filters` ConfigMap in the `tigera-operator` namespace
   with the following command.
   ```bash
   kubectl create configmap fluentd-filters -n tigera-operator --from-file=filters
   ```

### Manual/Helm deployment

DNS log filtering is configured in the `tigera-es-config` ConfigMap.

To enable DNS log filtering, follow these steps:

1. Set the `tigera.elasticsearch.dns-filtering` field in the `tigera-es-config`
   ConfigMap in the `calico-monitoring` Namespace to "true".  This ConfigMap can
   be found in `monitor-calico.yaml`.

1. Set the filters you wish to use in the `tigera.elasticsearch.dns-filters.conf`
   field.  See the following section for more information on writing those filters.

1. Force a rolling update of fluentd by patching the DaemonSet.
   ```bash
   kubectl patch daemonset -n calico-monitoring tigera-fluentd-node -p \
     "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"update-date\":\"`date +'%s'`\"}}}}}"
   ```

## Filter configuration files

The filters defined by the ConfigMap are inserted into the fluentd configuration file.
The [upstream fluentd documentation](https://docs.fluentd.org/filter/grep)
describes how to write fluentd filters.  The [DNS log schema](dns) can be referred to
for the specification of the various fields you can filter based on.  Remember to ensure
that the config file is properly indented in the ConfigMap.

### Example 1: filter out cluster-internal lookups

This example filters out lookups for domain names ending with ".cluster.local".  More
logs could be filtered by adjusting the regular expression "pattern", or by adding
additional `exclude` blocks.

```
<filter dns>
  @type grep
  <exclude>
    key qname
    pattern /\.cluster\.local$/
  </exclude>
</filter>
```

### Example 2: keep logs only for particular domain names

This example will filter out all logs *except* those for domain names ending `.co.uk`.

```
<filter dns>
  @type grep
  <regexp>
      key qname
      pattern /\.co\.uk$/
  </regexp>
</filter>
```
