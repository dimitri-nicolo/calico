#!/bin/bash

# Load CNX node image from archive.
podman load calico-early < /cnx-node.tar >&2

# Run CNX node in early networking mode.
podman run -d --privileged --net=host -v /calico-early:/calico-early -e CALICO_EARLY_NETWORKING=/calico-early/cfg.yaml --name calico-early calico-early >&2

while sleep 1; do
    if podman logs calico-early | grep "Early networking set up; now monitoring BIRD"; then
	break
    fi
done >&2

set - `ip -4 -o a show dev lo | grep 172.31.`
ipv4=${4%/*}
ipv6=fd5f:1234::$ipv4

echo $ipv4 $ipv6
