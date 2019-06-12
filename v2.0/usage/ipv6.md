---
title: IPv6 Support
canonical_url: https://docs.tigera.io/v2.3/usage/ipv6
---

{{site.prodname}} supports connectivity over IPv6, between compute hosts, and
between compute hosts and their containers. This means that, subject to
security configuration, a container can initiate an IPv6 connection to another
container, or to an IPv6 destination outside the data center; and that a
container can terminate an IPv6 connection from outside.

## Implementation details

Following are the key points of how IPv6 connectivity is currently
implemented in {{site.prodname}}.

-   IPv6 forwarding is globally enabled on each compute host.
-   Felix (the {{site.prodname}} agent):
    -   does `ip -6 neigh add lladdr dev`, instead of IPv4 case
        `arp -s`, for each endpoint that is created with an IPv6 address
    -   adds a static route for the endpoint's IPv6 address, via its tap
        or veth device, just as for IPv4
    -   configures Proxy NDP to ensure that routes to all machines go via
        the compute host.
-   BIRD6 runs between the compute hosts to distribute routes.
