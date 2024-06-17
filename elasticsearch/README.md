# Tigera elasticsearch

Upstream Elasticsearch for Tigera. This library modifies the image to remove CVEs and fix bugs in ECK or Elasticsarch that Elasticsearch has not fixed or has not back ported to the version of Elastcisearch or ECK that we are running.

## Upgrading Elasticsearch version

When upgrading Elasticsearch versions you MUST take note of the following:

* The packages installed in an Elasticsearch image may change in new versions, so the list of packages removed must be re-evaluated on upgrade.
* The readiness probe has been rewritten in golang and added to the pod (see `/readiness-probe`). You must make sure the ECK library version in go.mod matches the ECK version this will be deployed with, and the readiness probe must be re evaluated with the new ECK version and must match the bash script readiness probe for that ECK version. For instance, if this is to be deployed with ECK 2.5.0, check the readiness probe script for ECK 2.5.0 [here](https://github.com/elastic/cloud-on-k8s/blob/2.5.0/pkg/controller/elasticsearch/nodespec/readiness_probe.go#L33).
* The ECK version in the `go.mod` file must be updated to match the version of ECK this is deployed with.

### Update version and patch

To update Elasticsearch version, update the `ELASTIC_VERSION` variable in the `third_party/elasticsearch/Makefile`.

To update Tigera customization patch, follow the next steps:

1. Clone the upstream [Elasticsearch](https://github.com/elastic/elasticsearch) repository to your develop machine.
2. Switch to a target release tag.
3. Apply all patches under the [third_party/elasticsearch/patches](/third_party/elasticsearch/patches) folder to your clone and resolve conflicts.
4. Make your changes, and update dependencies by running `./gradlew --write-verification-metadata sha256 help`.
5. Use [`git commit`](https://git-scm.com/docs/git-commit) to commit your changes into your clone. It can be multiple commits and you don't need to push them.
6. Use [`git format-patch`](https://git-scm.com/docs/git-format-patch) to generate patch files. If you have multiple commits, you need to generate one patch file for each commit.
7. Copy patch files back to the [third_party/elasticsearch/patches](/third_party/elasticsearch/patches) folder and update the `patch` lines in `Makefile`.
8. Build a new elasticsearch image and validate.

You may also want to read the official [Building Elasticsearch with Gradle](https://github.com/elastic/elasticsearch/blob/main/BUILDING.md) guide.

### Building the image

Build third-party Elasticsearch image first and then run `make image` to create the production image.

### Releasing / Deploying on ECK

The readiness probe for ECK must be changed to use the one added to the image at `/readiness-probe`.
