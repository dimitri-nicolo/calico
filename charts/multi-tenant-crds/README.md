# Multi-tenancy Custom Resource Definition

This chart defines `crd.projectcalico.org` and `operator.tigera.io` CRDs needed for multi-tenancy.
These CRDs are tailored to be used on a multi-tenancy management cluster setup and different from the upstream version of the same CRDs used for a standalone or management Enterprise cluster.

## Tigera-operator chart

Installing tigera-operator helm chart will be done without installing the actual CRDs.

```
helm install calico-enterprise tigera-operator-{CALIENT_RELEASE_VERSION}.tgz --namespace tigera-operator \
--skip-crds \
--set manager.enabled=false \
--set policyRecommendation.enabled=false
```

## Retrieve latest version of multi-tenancy CRDs

In order to update multi-tenancy CRDs to the latest version, run the makefile target below:

```
OPERATOR_BRANCH=master make get-operator-crds
```

## Release a new chart

In order to create a new version of the chart, run the makefile target below:

```
RELEASE_STREAM=v3.18.1-ep2 make multi-tenant-crds-release
```

or

```
make chart
```

## Installing the chart

```
helm install calico-enterprise-crds multi-tenant-crds-{RELEASE_VERSION}.tgz
```

## Upgrading CRDs

```
helm template --include-crds  multi-tenant-crds-{RELEASE_VERSION}.tgz | kubectl apply --server-side --force-conflicts -f -
```
