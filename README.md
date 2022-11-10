# Tigera elasticsearch

Upstream Elasticsearch for Tigera. This library modifies the image to remove CVEs and fix bugs in ECK or Elasticsarch that 
Elasticsearch has not fixed or has not back ported to the version of Elastcisearch or ECK that we are running.

## Upgrading Elasticsearch version

When upgrading Elasticsearch versions you MUST take note of the following:
* The packages installed in an Elasticsearch image may change in new versions, so the list of packages removed must be 
* re-evaluated on upgrade. 
* The readiness probe has been rewritten in golang and added to the pod (see `/readiness-probe`). You must make sure the 
ECK library version in go.mod matches the ECK version this will be deployed with, and the readiness probe must be
re evaluated with the new ECK version and must match the bash script readiness probe for that ECK version. For instance,
if this is to be deployed with ECK 2.5.0, check the readiness probe script for ECK 2.5.0
[here](https://github.com/elastic/cloud-on-k8s/blob/2.5.0/pkg/controller/elasticsearch/nodespec/readiness_probe.go#L33).
* The ECK version in the `go.mod` file must be updated to match the version of ECK this is deployed with.

### Building the image

Run `make image` to create the image, run `make compressed-image` to create an image with the extra layers removed.

### Releasing / Deploying on ECK

The image released must be the compressed image, e.g. the one created by running `make compressed-image`. The readiness 
probe for ECK must be changed to use the one added to the image at /readiness-probe.