# Titan

Titan is a managed cluster scale testing tool designed to easily fake a Calico Cloud managed cluster and generate different types of log traffic (flows, events, audit, dns, etc) between a “Titan” managed cluster and a real management cluster hosted by Calico Cloud. Titan combines 2 different test tools together - [fake-guardian](../fake-guardian) by [Gordon](https://github.com/gcosgrave) and the [fake-log-generator](../fake-log-generator/) by [Lance](https://github.com/lwr20).

## Local Setup

Run `install_titan.sh`, and it will guide you through a series of steps to install titan.
