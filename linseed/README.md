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

| ENV                              |                         Default value                         |                                                                                           Description |
|----------------------------------|:-------------------------------------------------------------:|------------------------------------------------------------------------------------------------------:|
| LINSEED_PORT                     |                             `443`                             |                                                                       Local Port to start the service |
| LINSEED_HOST                     |                            <empty>                            |                                                                                  Host for the service |
| LINSEED_LOG_LEVEL                |                            `Info`                             |                                                                              Log Level across service |
| LINSEED_HTTPS_CERT               |                    `/certs/https/tls.crt`                     |                                                                                      Path to tls cert |
| LINSEED_HTTPS_KEY                |                    `/certs/https/tls.key`                     |                                                                                       Path tp tls key |
| LINSEED_FIPS_MODE_ENABLED        |                            `false`                            |                                               FIPSModeEnabled Enables FIPS 140-2 verified crypto mode |
| LINSEED_ELASTIC_ENDPOINT         | `https://tigera-secure-es-http.tigera-elasticsearch.svc:9200` |                                     Elastic Endpoint; For local development use http://localhost:9200 |
| LINSEED_ELASTIC_USERNAME         |                            <empty>                            |                 Elastic username; If left empty, communication with Elastic will not be authenticated |
| LINSEED_ELASTIC_PASSWORD         |                            <empty>                            |                 Elastic password; If left empty, communication with Elastic will not be authenticated |
| LINSEED_ELASTIC_CA_BUNDLE_PATH   |                `/certs/elasticsearch/tls.crt`                 |                                                                           Elastic ca certificate path |
| LINSEED_ELASTIC_CLIENT_CERT_PATH |               `/certs/elasticsearch/client.crt`               | Elastic client certificate path; It will only be picked up if LINSEED_ELASTIC_MTLS_ENABLED is enabled |
| LINSEED_ELASTIC_CLIENT_KEY_PATH  |               `/certs/elasticsearch/client.key`               |         Elastic client key path; It will only be picked up if LINSEED_ELASTIC_MTLS_ENABLED is enabled |
| LINSEED_ELASTIC_MTLS_ENABLED     |                            `false`                            |                                                Enables mTLS communication between Elastic and Linseed |
| LINSEED_ELASTIC_GZIP_ENABLED     |                            `false`                            |                                                Enables gzip communication between Elastic and Linseed |
| LINSEED_ELASTIC_SCHEME           |                            `https`                            |                                                  Defines what protocol is used to sniff Elastic nodes |
| LINSEED_ELASTIC_SNIFFING_ENABLED |                            `false`                            |                                                                    Enabled sniffing for Elastic nodes |


<!---
Describe what permissions needs in k8s cluster
--->


## Docs

- [Low level design for changes to the log storage subsystem](https://docs.google.com/document/d/1raHOohq0UWlLD9ygqsvu4vPMNNS9iGeY5xhHKt0O3Hc/edit?usp=sharing)
- [Multitenancy Proposal](https://docs.google.com/document/d/1HM0gba3hlR_cdTqHWc-NSqoiGHrVdTc_g1w3k8NmSdM/edit?usp=sharing)



