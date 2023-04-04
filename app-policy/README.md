# Application Layer Policy

Application Layer Policy for [Project Calico][calico] enforces network and
application layer authorization policies using [Istio].

![arch](docs/arch.png)

Istio mints and distributes cryptographic identities and uses them to establish mutually authenticated TLS connections
between pods.  Calico enforces authorization policy on this communication integrating cryptographic identities and 
network layer attributes.

The `envoy.ext_authz` filter inserted into the proxy, which calls out to Dikastes when service requests are
processed.  We compute policy based on a global store which is distributed to Dikastes by its local Felix.
 
## Getting Started

Application Layer Policy is described in the [Project Calico docs][docs].

 - [Enabling Application Layer Policy](https://docs.projectcalico.org/master/security/app-layer-policy)

 [calico]: https://projectcalico.org
 [istio]: https://istio.io
 [docs]: https://docs.projectcalico.org/latest
 

## Command-line options and environment variables

### Dikastes Server

    Usage:
      dikastes server [options]

    Options:
        -n --listen-network <net>   Listen network e.g. tcp, unix [default: unix]
        -l --listen <port>          Listen address [default: /var/run/dikastes/dikastes.sock]
        -x --dial-network <net>     PolicySync network e.g. tcp, unix [default: unix]
        -d --dial <target>          PolicySync address to dial to [default: localhost:50051]
        -r --rules <target>         Directory where WAF rules are stored. If this is specified, this runs dikastes with WAF rules processing.
        --log-level <level>         Log at specified level e.g. panic, fatal,info, debug, trace [default: info].

    Environment variables:

        DIKASTES_SUBSCRIPTION_TYPE
        
        sets the subscription type that's sent as a synchronization request to policy sync. valid values: per-pod-policies (default), per-host-policies.


        DIKASTES_ENABLE_CHECKER_REFLECTION
        
        this can be set to any value. if set, it enables grpc service reflection. Used for debugging / development.



