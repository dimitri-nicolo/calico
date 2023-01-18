# LINSEED

Linseed is a REST API for interacting with Calico Enterprise data primitives like flows, DNS logs, events, alerts and reports.

## Building and testing

To build this locally, use one of the following commands:

```
make image
```

or

```
make ci
```

## Adding and running test

To run all tests

```
make test
```

## Configuration and permissions

| ENV                |     Default value      |            Description |
|--------------------|:----------------------:|-----------------------:|
| LINSEED_PORT        |         `443`          | Local Port to start the service |
| LINSEED_HOST        |        <empty>         |   Host for the service |
| LINSEED_LOG_LEVEL     |         `Info`         |    Log Level across service |
| LINSEED_CERT        | `/certs/https/tls.crt` |    Path to tls cert |
| LINSEED_KEY        | `/certs/https/tls.key` |    Path tp tls key |
| LINSEED_FIPS_MODE_ENABLED |        `false`         |    FIPSModeEnabled Enables FIPS 140-2 verified crypto mode |

<!---
Describe what permissions needs in k8s cluster
--->


## Docs

- [Low level design for changes to the log storage subsystem](https://docs.google.com/document/d/1raHOohq0UWlLD9ygqsvu4vPMNNS9iGeY5xhHKt0O3Hc/edit?usp=sharing)
- [Multitenancy Proposal](https://docs.google.com/document/d/1HM0gba3hlR_cdTqHWc-NSqoiGHrVdTc_g1w3k8NmSdM/edit?usp=sharing)



