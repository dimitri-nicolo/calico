# LINSEED

Linseed is a REST API for interacting with Calico Enterprise data primitives like:
- flows and flow logs
- DNS flows and DNS logs
- L7 flows and L7 logs
- audit logs (Kubernetes and Calico Enterprise)
- bgp logs
- events
- runtime reports

It also provided additional APIs that extract high level information from the primitives described above. 
- processes (extracted from flow logs)

*What defines a log ?*

A log records the raw event that happened when two components interact within a K8S cluster over a period of time.
An example of such interaction can be establishing the connection between a client application and server application.
A log can gather statistics like how much data is being transmitted, who initiated the interaction and if the interaction was successful or not.

Logs have multiple flavours, as they gather raw data at L3-L4 level, L7 level, DNS, K8s Audit Service, BGP etc.

*What defines a flow ?*

A flow is an aggregation of one or multiple logs that describe the interaction between a source and destination over a given period of time.
Flows have direction, as they can be reported by either the source and destination.


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

In order to run locally, start an elastic server on localhost using:

```
make run-elastic
```

Start Linseed with the following environment variables:

- LINSEED_ELASTIC_ENDPOINT=http://localhost:9200
- LINSEED_HTTPS_CERT=~/calico-private/linseed/fv/cert/localhost.crt
- LINSEED_HTTPS_KEY=~/calico-private/linseed/fv/cert/localhost.key

Or simply use the following command:

```
make run-image
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


Linseed is deployed in namespace `tigera-elasticsearch` as part of Calico Enterprise installation.
It establishes connections with the following components:
- `tigera-elasticsearch/tigera-elasticsearch` pod via service `tigera-secure-es-http.tigera-elasticsearch.svc:9200`

It has the following clients, via service `tigera-linseed.tigera-elasticsearch.svc`
- `es-proxy` container from `tigera-manager/tigera-manager` pod
- `intrusion-detection-controller` container from `tigera-intrusion-detection/intrusion-detection-controller` pod
- `fluentd-node` container from `tigera-fluentd/fluentd-node` pod

It requires RBAC access for CREATE for authorization.k8s.io.SubjectAccessReview at namespace level.
X509 certificates will be mounted inside the pod via operator at `/etc/pki/tls/certs/` and `/tigera-secure-linseed-cert`


## Docs

- [Low level design for changes to the log storage subsystem](https://docs.google.com/document/d/1raHOohq0UWlLD9ygqsvu4vPMNNS9iGeY5xhHKt0O3Hc/edit?usp=sharing)
- [Multi-tenancy Proposal](https://docs.google.com/document/d/1HM0gba3hlR_cdTqHWc-NSqoiGHrVdTc_g1w3k8NmSdM/edit?usp=sharing)



