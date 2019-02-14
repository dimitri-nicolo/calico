| Field       | Description                                | Accepted Values                                                   | Schema                    | Default    |
|-------------|--------------------------------------------|-------------------------------------------------------------------|---------------------------|------------|
| action      | Action to perform when matching this rule. | `Allow`, `Deny`, `Log`, `Pass`                                    | string                    |            |
| protocol    | Positive protocol match.                   | `TCP`, `UDP`, `ICMP`, `ICMPv6`, `SCTP`, `UDPLite`, `1`-`255`      | string \| integer         |            |
| notProtocol | Negative protocol match.                   | `TCP`, `UDP`, `ICMP`, `ICMPv6`, `SCTP`, `UDPLite`, `1`-`255`      | string \| integer         |            |
| icmp        | ICMP match criteria.                       |                                                                   | [ICMP](#icmp)             |            |
| notICMP     | Negative match on ICMP.                    |                                                                   | [ICMP](#icmp)             |            |
| ipVersion   | Positive IP version match.                 | `4`, `6`                                                          | integer                   |            |
| source      | Source match parameters.                   |                                                                   | [EntityRule](#entityrule) |            |
| destination | Destination match parameters.              |                                                                   | [EntityRule](#entityrule) |            |
| allowedEgressDomains | Domain name match (see below).    | List of domain names.                                             | [ string, ... ]           |            |
| http        | Match HTTP request parameters. Application layer policy must be enabled to use this field. |                   | [HTTPMatch](#httpmatch)   |            |

An `action` of `Pass` will skip over the remaining policies and jump to the
first [profile]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/profile) assigned to the endpoint, applying the policy configured in the
profile; if there are no Profiles configured for the endpoint the default
applied action is `Deny`.

`allowedEgressDomains` is an optional field, valid for Allow rules only, that restricts the rule to
apply only to traffic to one of the specified domains.  If this field is specified, `action` must be
`Allow`, and `destination.nets` and `destination.selector` must both be left empty.
