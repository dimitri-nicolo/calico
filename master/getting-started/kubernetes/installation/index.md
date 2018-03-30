---
title: Installing CNX on Kubernetes
---

{{site.prodname}} can be installed on a Kubernetes cluster in a number of configurations.  This document
gives an overview of the most popular approaches, and provides links to each for more detailed
information.

## [{{site.prodname}} Hosted Install](hosted)

Installs the {{site.prodname}} components as a DaemonSet entirely using Kubernetes manifests through a single
kubectl command. 

## [Custom Installation](integration)

In addition to the hosted approach above, the {{site.prodname}} components can also be installed using your
own orchestration mechanisms (e.g ansible, chef, bash, etc)

Follow the [integration guide](integration) if you would like
to integrate {{site.prodname}} into your own installation or deployment scripts.
