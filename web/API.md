## calicoqweb API Documentation

This API documentation is not meant for external/public consumption and is
targeted at internal use (cnx-manager/webapp only).

### Available APIS

* [Version](#version)
* [Summary](#summary)
* [Policies](#policies)
  * [Global Network Policy](#globalnetworkpolicies)
  * [Network Policy](#networkpolicies)
* [Endpoints](#endpoints)
  * [Workload Endpoints](#workload-endpoints)
  * [Host Endpoints](#host-endpoints)
* [Nodes](#nodes)

### General API Principles

1. All APIS are read-only and *only* support the GET method.
1. There is currently no pagination available. The API will return the first
   `n` matching policies. Because of this behaviour, all APIs will return a
   `count` field that that gives the total count and the actual entries will
    be limited to `n`.

TODO(doublek):
1. What is the value of `n`.

### Version

Returns the version of `calicoqweb`.

```
http://host:port/base/version
```

### Summary

Retrieve a statistics summary of policies, endpoints, and nodes.

UI Req:
1. Intended to be used from the dashboard view. In addition to the
packet/connection statistics, the dashboard contains a panel to show total
policies/endpoints and nodes.

```
http://host:port/base/summary
```

TODO(doublek):
1. The dashboard requires unused policy count and denying policy count. We could include unused here?

#### Query Parameters

There are currently no supported query parameters for the `summary` API.

#### Response

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| numGlobalNetworkPolicies | Count of globalnetworkpolicies | number |
| numNetworkPolicies | Count of (namespaced) networkpolicies | number |
| numWorkloadEndpoints | Count of workload endpoints | number |
| numUnlabelledWorkloadEndpoints | Count of workload endpoints that do not have a label associated | number |
| numUnlabelledHostEndpoints | Count of host endpoints that do not have a label associated | number |
| numNodesWithNoHostEndpoints | Count of nodes that do not have any host endpoints configured | number |
| numHostEndpoints | Count of host endpoints | number |
| numNodes | Count of nodes | number |

NOTE:
1. To get the total endpoint count to for this policy, sum `numWorkloadEndpoints` and `numHostEndpoints`.
1. `numNodesWithNoHostEndpoints` is loosely equivalent to "Unprotected Nodes"

### Policies

UI Req:
1. Intented to be used in the policies view. The web UI currently has a policy
view. In this view there is a choice of "trello" style lists, or a tabular view.
Both the views show endpoint counts per policy. The tabular view can expand a
policy to show some additional details about the policy. Most of the information
is already obtained from the CNX-Apiserver and for each policy, this API should
provide the number of endpoints and nodes.
1. Intended to be used in a policy details view. This view shows a single policy
and other fields of a policy including ingress and egress rules. This view shows
the connection statistics and endpoint statistics. This API should provice the
number of endpoints matching a policy and number of endpoints matching each
rule.
1. For both the above requirements, the data provided is not based on actual
traffic seen by the system but by examining policy selectors and such.

Design Note:
1. The API is split into two, globalnetworkpolicies and networkpolicies to keep
   close with how the AAPI server provides APIs.
1. However, in the calicoqweb implementation, we will not provide a per namespace
   resource API. Instead the namespace will be included as part of the query
   parameter.

#### GlobalNetworkPolicies

```
http://host:port/base/globalnetworkpolicies/{name}
```

- `{name}` is optional and omitting it means return all GlobalNetworkPolicies

##### Query Parameters

| Name | Description | Type | Repeated | Required |
| ---- | ----------- | ---- | -------- | -------- |
| tier | Get globalnetworkpolicies that are in a tier | string | no | no |
| unmatched | Get globalnetworkpolicies whose selectors do not match any endpoints | boolean | no | no |
| workloadEndpoint | Get globalnetworkpolicies that match a workload endpoint | string | no | no |
| hostEndpoint | Get globalnetworkpolicies that match a host endpoint | string | no | no |
| selector | Get globalnetworkpolicies that match a selector | [selector expression](#selectors) | yes | no |

- When no query parameter is provided, results should be returned for all tiers.
- Multiple query parameters can be combined together (read exceptions below) and
  they will be treated as a logical AND. Results matching all the query
  parameters should be returned.
- The `workloadEndpoint` and `hostEndpoint` query parameters cannot be combined
  in a single query.
- When the `tier` is specified, results will be limited to the specified `tier`.

##### Selectors

The query parameter `selector` has to be in the policy selector format (Refer
to policy resource docs).
The `selector` field can also be repeated multiple times and they will be
combined using logical AND (`&&`). Only policies that match all provided
selectors will be returned.

#### Response

TODO(doublek):
1. Note on sorting order

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| count | Count of policies matching the request | number |
| items | A list of globalnetworkpolicies that match the query | list of [global network policy response objects](#global-network-policy-response-object)|

##### Global Network Policy Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| name | The name of the globalnetworkpolicy | string |
| tier | The tier the globalnetworkpolicy belongs to | string |
| numWorkloadEndpoints | The number of workload endpoints matching the globalnetworkpolicy | number |
| numHostEndpoints | The number of host endpoints matching the globalnetworkpolicy | number |
| numNodes | The number of nodes this globalnetworkpolicy is applied | number |
| ingressRules | List of ingress rules | list of [rule](#rule-response-object) |
| egressRules | List of egress rules | list of [rule](#rule-response-object) |

NOTE:
1. The `name` parameter is exactly the same as in the v3 client.
   - It is prefixed with the tier name.
1. To get the total endpoint count to for this policy, sum
   `numWorkloadEndpoints` and `numHostEndpoints`.

##### Rule Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| numWorkloadEndpoints | The number of workload endpoints matching the rule selector (if any) | number |
| numHostEndpoints | The number of host endpoints matching the rule selector (if any) | number |

#### NetworkPolicies

```
http://host:port/base/networkpolicies/{name}
```

- `{name}` is optional and omitting it means return all NetworkPolicies
  (including Kubernetes Network Policies).
- When provided `{name}` should be prefixed with `knp.default.` for
  Kubernetes Network Policy or with the correct tier name.

#### Query Parameters

| Name | Description | Type | Repeated | Required |
| ---- | ----------- | ---- | -------- | -------- |
| namespace | Get networkpolicies that are in a namespace | string | no | no |
| unused | Get unused networkpolicies | boolean | no | no |
| workloadEndpoint | Get networkpolicies that match a workload endpoint | string | no | no |
| hostEndpoint | Get networkpolicies that match a host endpoint | string | no | no |
| selector | Get networkpolicies that match a selector | [selector expression](#selectors) | yes | no |

- When no query parameter is provided, results should be returned for all tiers and all namespaces.
- Multiple query parameters can be combined together and they will be treated
  as a logical AND. Results matching all the query parameters should be
  returned.
- The `workloadEndpoint` and `hostEndpoint` query parameters cannot be combined
  in a single query.
- When the `tier` is specified, results will be limited to the specified `tier`.

- TODO(doublek): What to do when namespace is not present in the query parameter.
  Do we return all or default to `default`. In the latter case, then namespace
  should be in the path.

#### Response

TODO(doublek):
1. Note on sorting order

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| count | Count of policies matching the request | number |
| items | A list of policies that match the query | list of [network policy response objects](#network-policy-response-object)|

##### Network Policy Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| name | The name of the policy | string |
| namespace | The namespace the policy belongs to | string |
| tier | The tier the policy belongs to | string |
| numWorkloadEndpoints | The number of workload endpoints matching the policy | number |
| numHostEndpoints | The number of host endpoints matching the policy | number |
| numNodes | The number of nodes this policy is applied | number |
| ingressRules | List of ingress rules | list of [rule](#rule-response-object) |
| egressRules | List of egress rules | list of [rule](#rule-response-object) |

NOTE:
1. The `name` parameter is exactly the same as in the v3 client.
   - It is prefixed with `knp.default.` for Kubernetes network policy,
   - It is prefixed with the tier name otherwise.
1. To get the total endpoint count to for this policy, sum
   `numWorkloadEndpoints` and `numHostEndpoints`.

##### Rule Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| numWorkloadEndpoints | The number of workload endpoints matching the rule selector (if any) | number |
| numHostEndpoints | The number of host endpoints matching the rule selector (if any) | number |

#### Example

TODO(doublek)

### Endpoints

UI Req:
1. Intended to be used in a endpoints view, which displays a list of endpoints (both host
and workload). This API should list all endpoints, with the ability to filter by label or
filter by a policy.
1. Clicking on an endpoint expands the row in the table and displays some detailed information
about the endpoint such as IP addresses, labels and all policies that apply to this endpoint.

Design Note:
1. The API is split into two, workloadendpoints and hostendpoints to keep
   close with how the AAPI server may provide APIs in the future.
1. However, in the calicoqweb implementation, we will not provide a per namespace
   resource API. Instead the namespace will be included as part of the query
   parameter.
1. Clicking on a endpoint, expands to provide additional details about the node
   such as endpoints and IP addresses. On this event, the web client is
   expected to issue [endpoints](#endpoints) query with the appropriate
   `node` query parameter filled in.

#### Workload Endpoints

```
http://host:port/base/workloadendpoints/{name}
```

- `{name}` is optional and omitting it means return all WorkloadEndpoints matching any other query parameter.

TODO(doublek):
1. Note on sorting order - instead of returning endpoints, should we return
   a list of nodes, each containing endpoints and hence grouping by nodes?

#### Query Parameters

| Name | Description | Type | Repeated | Required |
| ---- | ----------- | ---- | -------- | -------- |
| namespace | Get endpoints that are in a namespace | string | no | no |
| node | Get endpoints that the endpoint resides in | string | no | no |
| policy | Get endpoints that the a policy applies on | string | no | no |
| selector | Get workloadendpoints that match a selector | [selector expression](#selectors) | yes | no |
| namespaceSelector | Get workloadendpoints that belong in a namespace | [selector expression](#selectors) | yes | no |

- When no query parameter is provided, results should be returned for all namespaces.
- Multiple query parameters can be combined together and they will be treated
  as a logical AND. Results matching all the query parameters should be
  returned.
- `namespaceSelector` is meant to be used when dealing with `NetworkPolicy` rules.

#### Response

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| count | Count of workloadendpoints matching the request | number |
| items | A list of workloadendpoints that match the query | list of [workloadendpoint response objects](#workload-endpoint-response-object) |

##### Workload Endpoint Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| name | The name of the the endpoint | string |
| workload | The name of the workload to which this endpoint belongs | string |
| orchestrator | The orchestrator that created this endpoint | string |
| pod | The kubernetes pod name (if orchestrator value is `k8s`) | string |
| ipNetworks | List of CIDRs assigned to this endpoint | list of strings |
| labels | List of labels that applies to this endpoint | list of key-value pairs |
| node | The node that the endpoint resides in | string |
| interfaceName | The name of the interface attached to this endpoint | string |
| numPolicies | The number of policies that are applied to this endpoints | number |

#### Example

TODO(doublek)

#### Host Endpoints

```
http://host:port/base/hostendpoints/{name}
```

- `{name}` is optional and omitting it means return all HostEndpoints (filtered by any other query parameter).

#### Query Parameters

| Name | Description | Type | Repeated | Required |
| ---- | ----------- | ---- | -------- | -------- |
| node | Get hostendpoints that the resides on a node | string | no | no |
| policy | Get hostendpoints that the a policy applies on | string | no | no |
| selector | Get hostendpoints that match a selector | [selector expression](#selectors) | yes | no |

- When no query parameter is provided, results should returned all host endpoints.
- Multiple query parameters can be combined together and they will be treated
  as a logical AND. Results matching all the query parameters should be
  returned.

#### Response

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| count | Count of hostendpoints matching the request | number |
| items | A list of hostendpoints that match the query | list of [hostendpoint response objects](#host-endpoint-response-object) |

##### Host Endpoint Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| name | The name of the the endpoint | string |
| labels | List of labels that applies to this endpoint | list of key-value pairs |
| expectedIps | List of expected IPs associated with this interface | list of strings |
| node | The node that the endpoint resides in | string |
| interfaceName | The name of the interface attached to this endpoint | string |
| numPolicies | The number of policies that are applied to this endpoints | number |

#### Example

TODO(doublek)

### Nodes

UI Req:
1. Intended to be used in the nodes view, which displays a list of nodes. This API should
list all nodes, with the ability to filter by policies that are applied to endpoints on a
node.
1. Each node contains the number of endpoints that reside on each node.

Design Note:
1. The webapp could get a list of nodes via the Kubernetes API, and simply use
   the calicoq API for filling in counts? (TBD)
1. Clicking on a node, expands to provide additional details about the node
   such as endpoints and IP addresses. On this event, the web client is
   expected to issue [endpoints](#endpoints) query with the appropriate
   `node` query parameter filled in.

```
http://host:port/base/nodes/{name}
```

- `{name}` is optional and omitting it means return all Nodes

#### Query Parameters

There are currently no supported query parameters for the `nodes` API.

#### Response

Returns a JSON object with the following fields.

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| count | Count of nodes matching the request | number |
| items | A list of nodes that match the query | list of [node response objects](#node-response-object) |

##### Node Response Object

| Field | Description | Scheme |
| ----- | ----------- | ------ |
| name | Name of the node | string |
| numWorkloadEndpoints | The number of workload endpoints residing on this node | number |
| numHostEndpoints | The number of host endpoints present on this node | number |

#### Example

TODO(doublek)
