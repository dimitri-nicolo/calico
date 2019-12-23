---
title: Filtering out DNS logs
canonical_url: https://docs.tigera.io/master/security/logs/elastic/filtering-dns
---

{{site.tseeprodname}} supports filtering out DNS logs based on user provided
configuration.  This functionality is intended to be used to suppress logs
of low significance.

## Configuring DNS log filtering

DNS log filtering is configured in the `tigera-es-config` ConfigMap.

To enable DNS log filtering, follow these steps.

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

## Writing filter configuration files

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
