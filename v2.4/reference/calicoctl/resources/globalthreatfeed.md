---
title: Global Threat Feed Resource (GlobalThreatFeed)
redirect_from: latest/reference/calicoctl/resources/globalthreatfeed
---

A global threat feed resource (GlobalThreatFeed) represents a feed of threat intelligence used for
security purposes.

At present, {{site.prodname}} supports threat feeds that give a set of IP addresses
or IP prefixes, and automatically monitors flow logs for members of the set. These threat feeds
have their content type set to IPSet.  IPSet threat feeds can also be configured to be synchronized to a
[global network set](./globalnetworkset), allowing you to use them as a dynamically-updating 
blacklist by incorporating the global network set into network policy.

For `calicoctl` [commands]({{site.url}}/{{page.version}}/reference/calicoctl/commands/), the following case-insensitive aliases 
may be used to specify the resource type on the CLI:
`globalthreatfeed`, `globalthreatfeeds`.

For `kubectl` [commands](https://kubernetes.io/docs/reference/kubectl/overview/), the following case-insensitive aliases 
may be used to specify the resource type on the CLI:
`globalthreatfeed.projectcalico.org`, `globalthreatfeeds.projectcalico.org` and abbreviations such as 
`globalthreatfeed.p` and `globalthreatfeeds.p`.

### Sample YAML

```yaml
apiVersion: projectcalico.org/v3
kind: GlobalThreatFeed
metadata:
  name: sample-global-threat-feed
spec:
  content: IPSet
  globalNetworkSet:
    # labels to set on the GNS
    labels:
      level: high
  pull:
    # accepts time in golang duration format
    period: 24h
    http:
      format: NewlineDelimited
      url: https://an.example.threat.feed/blacklist
      headers:
        - name: "Accept"
          value: "text/plain"
        - name: "APIKey"
          valueFrom:
            # secrets selected must be in the 
            # "calico-monitoring" namespace in order
            # to be used
            secretKeyRef:
              name: "example-threat-feed"
              key: "apikey"
```

### Push or Pull
You can configure {{site.prodname}} to pull updates from your threat feed using a [`pull`](#pull) stanza in 
the global threat feed spec.

Alternately, you can have your threat feed push updates directly.  Leave out the `pull` stanza, and configure
your threat feed to create or update the Elasticsearch document that corresponds to the global threat
feed object.  This Elasticsearch document will be in the index `.tigera.ipset.<cluster_name>` and must have the ID set
to the name of the global threat feed object. The doc should have a single field called `ips`, containing
a list of IP prefixes.

For example:

```
PUT .tigera.ipset.cluster01/_doc/sample-global-threat-feed
{
    "ips" : ["99.99.99.99/32", "100.100.100.0/24"]
}
```

Refer to the [Elasticsearch document APIs][elastic-document-apis] for more information on how to 
create and update documents in Elasticsearch.


### GlobalThreatFeed Definition

#### Metadata

| Field       | Description                                   | Accepted Values   | Schema  |
|-------------|-----------------------------------------------|-------------------|---------|
| name        | The name of this threat feed.                 | Lower-case alphanumeric with optional `-`  | string  |
| labels      | A set of labels to apply to this threat feed. |                   | map     |

#### Spec

| Field            | Description                                    | Accepted Values | Schema                                        | Default |
|------------------|------------------------------------------------|-----------------|-----------------------------------------------|---------|
| content          | What kind of threat intelligence is provided   | IPSet           | string                                        | IPSet   |
| globalNetworkSet | Include to sync with a global network set      |                 | [GlobalNetworkSetSync](#globalnetworksetsync) |         |
| pull             | Configure periodic pull of threat feed updates |                 | [Pull](#pull)                                 |         |

`IPSet` is the only supported `content` type at present, and is a list of IP addresses or IP prefixes.

#### Status

The `status` is read-only for users and updated by the `intrusion-detection-controller` component as
it processes global threat feeds.

| Field                | Description                                                                      |
|----------------------|----------------------------------------------------------------------------------|
| lastSuccessfulSync   | Timestamp of the last successful update to the threat intelligence from the feed |
| lastSuccessfulSearch | Timestamp of the last successful search of flow logs for threats                 |
| errorConditions      | List of errors preventing operation of the updates or search                     |

#### GlobalNetworkSetSync

When you include a `globalNetworkSet` stanza in a global threat feed, it triggers synchronization
with a [global network set](./globalnetworkset). This global network set will have the name `threatfeed.<threat feed name>`
where `<threat feed name>` is the name of the global threat feed it is synced with.

> **NOTE**: If you include a `globalNetworkSet` stanza, you must also include a `pull` stanza.
{: .alert .alert-info}

| Field  | Description                                               | Accepted Values | Schema |
|--------|-----------------------------------------------------------|-----------------|--------|
| labels | A set of labels to apply to the synced global network set |                 | map    |

#### Pull

When you include a `pull` stanza in a global threat feed, it triggers a periodic pull of new data. On successful
pull and update to the data store, we update the `status.lastSuccessfulSync` timestamp.

If you do not include a `pull` stanza, you must configure your system to [push](#push-or-pull) updates. 

| Field  | Description                           | Accepted Values | Schema                            | Default |
|--------|---------------------------------------|-----------------|-----------------------------------|---------|
| period | How often to pull an update           | >= 5m           | [Duration string][parse-duration] | 24h     |
| http   | Pull the update from an HTTP endpoint |                 | [HTTPPull](#httppull)             |         |

#### HTTPPull

Pull updates from the threat feed by doing an HTTP GET against the given URL.

| Field   | Description                                               | Accepted Values  | Schema                    | Default          |
|---------|-----------------------------------------------------------|------------------|---------------------------|------------------|
| format  | Format of the data the threat feed returns                | NewlineDelimited | string                    | NewlineDelimited |
| url     | The URL to query                                          |                  | string                    |                  |
| headers | List of additional HTTP Headers to include on the request |                  | [HTTPHeader](#httpheader) |                  |

The `format` must be set to `NewlineDelimited` or omitted.  The threat feed must return a list of
newline-delimited IP addresses or IP prefixes. It may also include comments prefixed by `#`.  For
example:

```
# This is an IP Prefix
100.100.100.0/24
# This is an address
99.99.99.99
```

#### HTTPHeader

| Field     | Description                                               | Schema                                |
|-----------|-----------------------------------------------------------|---------------------------------------|
| name      | Header name                                               | string                                |
| value     | Literal value                                             | string                                |
| valueFrom | Include to retrieve the value from a config map or secret | [HTTPHeaderSource](#httpheadersource) |

> **NOTE**: You must include either `value` or `valueFrom`, but not both.
{: .alert .alert-info}


#### HTTPHeaderSource

| Field           | Description                     | Schema            |
|-----------------|---------------------------------|-------------------|
| configMapKeyRef | Get the value from a config map | [KeyRef](#keyref) |
| secretKeyRef    | Get the value from a secret     | [KeyRef](#keyref) | 


#### KeyRef

KeyRef tells {{site.prodname}} where to get the value for a header.  The referenced Kubernetes object
(either a config map or a secret) must be in the `calico-monitoring` namespace.

| Field    | Description                                               | Accepted Values | Schema | Default |
|----------|-----------------------------------------------------------|-----------------|--------|---------|
| name     | The name of the config map or secret                      |                 | string |         |
| key      | The key within the config map or secret                   |                 | string |         |
| optional | Whether the pull can proceed without the referenced value | If the referenced value does not exist, `true` means omit the header. `false` means abort the entire pull until it exists | bool | `false`


[elastic-document-apis]: https://www.elastic.co/guide/en/elasticsearch/reference/6.4/docs-update.html
[parse-duration]: https://golang.org/pkg/time/#ParseDuration