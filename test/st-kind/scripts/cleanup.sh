#!/usr/bin/env bash

kind delete cluster
unset KUBECONFIG
rm -rf $(pwd)/test-resources
rm -rf /tmp/kind-configs
