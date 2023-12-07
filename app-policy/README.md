# Application Layer Policy

Application Layer Policy for [Project Calico][calico] enforces network and application layer authorization policies via Envoy External Authorization.

The `envoy.ext_authz` filter inserted into the proxy, which calls out to Dikastes when service requests are
processed.  We compute policy based on a global store which is distributed to Dikastes by its local Felix.

## Command-line options and environment variables

### Dikastes Server

      $ dikastes -h

      Usage of dikastes:
      -dial string
            PolicySync address (default "/var/run/nodeagent/socket")
      -dial-network string
            PolicySync network e.g. tcp, unix (default "unix")
      -http-server-addr string
            HTTP server address (default "0.0.0.0")
      -http-server-port string
            HTTP server port
      -listen string
            Listen address (default "/var/run/dikastes/dikastes.sock")
      -listen-network string
            Listen network e.g. tcp, unix (default "unix")
      -log-level string
            Log at specified level e.g. panic, fatal,info, debug, trace (default "info")
      -subscription-type string
            Subscription type e.g. per-pod-policies, per-host-policies (default "per-host-policies")
      -waf-directive value
            Additional directives to specify for WAF (if enabled). Can be specified multiple times.
      -waf-enabled
            Enable WAF.
      -waf-log-file string
            WAF log file path. e.g. /var/log/calico/waf/waf.log
      -waf-ruleset-base-dir string
            Base directory for WAF rulesets. (default "/")

    Environment variables:

    DIKASTES_SUBSCRIPTION_TYPE
    
      sets the subscription type that's sent as a synchronization request to policy sync. valid values: per-pod-policies (default), per-host-policies.

    DIKASTES_ENABLE_CHECKER_REFLECTION
    
      this can be set to any value. if set, it enables grpc service reflection. Used for debugging / development.



