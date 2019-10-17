| Field       | Description                 | Accepted Values   | Schema | Default    |
|-------------|-----------------------------|-------------------|--------|------------|
| nets                  | Match packets with IP in any of the listed CIDRs. | List of valid IPv4 or IPv6 CIDRs  | list of cidrs |
| notNets               | Negative match on CIDRs. Match packets with IP not in any of the listed CIDRs. | List of valid IPv4 or IPv6 CIDRs  | list of cidrs |
| selector    | Positive match on selected endpoints. If a `namespaceSelector` is also defined, the set of endpoints this applies to is limited to the endpoints in the selected namespaces. | Valid selector | [selector](#selector) | |
| notSelector | Negative match on selected endpoints. If a `namespaceSelector` is also defined, the set of endpoints this applies to is limited to the endpoints in the selected namespaces. | Valid selector | [selector](#selector) | |
| namespaceSelector | Positive match on selected namespaces. If specified, only workload endpoints in the selected Kubernetes namespaces are matched. Matches namespaces based on the labels that have been applied to the namespaces. Defines the context that selectors will apply to, if not defined then selectors apply to the NetworkPolicy's namespace. | Valid selector | [selector](#selector) | |
| ports | Positive match on the specified ports | | list of [ports](#ports) | |
| domains | Positive match on [domain names](#exact-and-wildcard-domain-names). | List of [exact or wildcard domain names](#exact-and-wildcard-domain-names) | list of strings |
| notPorts | Negative match on the specified ports | | list of [ports](#ports) | |
| serviceAccounts | Match endpoints running under service accounts. If a `namespaceSelector` is also defined, the set of service accounts this applies to is limited to the service accounts in the selected namespaces. | | [ServiceAccountMatch](#serviceaccountmatch) | |

#### Exact and wildcard domain names

The `domains` field is only valid for egress Allow rules.  It restricts the
rule to apply only to traffic to one of the specified domains.  If this field is specified, the
parent [Rule](#rule)'s `action` must be `Allow`, and `nets` and `selector` must both be left empty.

{% include {{page.version}}/domain-names.md %}

> **Note**: {{site.prodname}} implements policy for domain names by learning the
> corresponding IPs from DNS, then programming rules to allow those IPs.  This means that
> if multiple domain names A, B and C all map to the same IP, and there is domain-based
> policy to allow A, traffic to B and C will be allowed as well.
{: .alert .alert-info}
