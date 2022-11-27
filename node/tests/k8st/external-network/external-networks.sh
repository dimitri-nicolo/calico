#!/bin/bash -ex

# test directory.
TEST_DIR=./tests/k8st

# kubectl binary.
: ${kubectl:=./kubectl}

# kind binary.
: ${KIND:=dist/kind}

# k8s version.
: ${K8S_VERSION:=v1.24.7}

echo "Don't Download kind executable with multiple networks support"
#curl -L https://github.com/projectcalico/kind/releases/download/multiple-networks-0.3/kind -o ${KIND}
chmod +x ${KIND}

# Set config variables needed for ${kubectl} and calicoctl.
export KUBECONFIG=~/.kube/kind-config-kind

# Normally, cleanup any leftover state, then setup, then test.
: ${STEPS:=cleanup setup}

# URL for an operator install.
: ${OPERATOR_URL:=https://docs.tigera.io/master/manifests/tigera-operator.yaml}

# Full name and tag of the cnx-node image that the preceding URL uses.
# We need this because we will build the local node code into an image
# and then retag it - inside the test cluster - with exactly this
# name.  Then when the operator install proceeds it will pick up that
# image instead of pulling from gcr.io.
: ${CNX_NODE_IMAGE:=gcr.io/unique-caldron-775/cnx/tigera/cnx-node:master}

tmpd=$(mktemp -d -t calico.XXXXXX)


function add_calico_resources() {
    ${CALICOCTL} apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-a1
spec:
  nodeSelector: "egress == 'true'"
  peerIP: 172.31.11.1
  asNumber: 64512
  sourceAddress: None
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-b1
spec:
  nodeSelector: "egress == 'true'"
  peerIP: 172.31.21.1
  asNumber: 64512
  sourceAddress: None
---
EOF

    # Label and annotate nodes.
    ${kubectl} label node kind-worker egress=true --overwrite
    ${kubectl} annotate node kind-worker projectcalico.org/IPv4Address=172.18.0.4/24 --overwrite
}

# IP addressing scheme: 172.31.X.Y where
#
#   X = 10 * RACK_NUMBER + PLANE_NUMBER
#
#   Y = NODE_NUMBER (within rack)
#
# Networks BETWEEN racks have RACK_NUMBER = 0.
#
# Loopback addresses have PLANE_NUMBER = 0.

function do_setup {
    # Fix rp_filter setting.
    sudo sysctl -w net.ipv4.conf.all.rp_filter=1

    # Create docker networks for this topology:
    #
    #
    #                         'uplink'
    #    +---------------+    172.31.1   +---------------+
    #    | ToR (bird-a1) |---------------| ToR (bird-b1) |
    #    +---------------+ .2         .3 +---------------+
    #           |.1                             |.1
    #           |                               |
    #           |                               |
    #           | 172.31.11                     | 172.31.21
    #           |  'enetA'                        |  'enetB'
    #           |                               |
    #           |.3 .4                          |.3 .4
    #  +-----------------+             +-----------------+
    #  | Nodes of rack A |             | Nodes of rack B |
    #  +-----------------+             +-----------------+
    #     kind-control-plane              kind-worker2
    #     kind-worker                     kind-worker3
    # docker network create --subnet=172.18.0.0/24 --ip-range=172.18.0.0/24 --gateway 172.18.0.2 internal
    docker network create --subnet=172.31.11.0/24 --ip-range=172.31.11.0/24 --gateway 172.31.11.2 enetA
    docker network create --subnet=172.31.21.0/24 --ip-range=172.31.21.0/24 --gateway 172.31.21.2 enetB
    docker network create --subnet=172.31.31.0/24 --ip-range=172.31.31.0/24 --gateway 172.31.31.2 enetC

    # Create routers for external networks.
    docker run -d --privileged --net=enetA --ip=172.31.11.1 --name=bird-a1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.1 --name=bird-b1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.3 --name=bird-b2 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetC --ip=172.31.31.1 --name=bird-c1 ${ROUTER_IMAGE}

    docker network connect --ip=172.31.11.4 enetA kind-worker
    docker network connect --ip=172.31.21.4 enetB kind-worker
    docker network connect --ip=172.31.31.4 enetC kind-worker

    # Configure Router end of cluster node peerings.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/nodes-enetA.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export all;
  add paths on;
  graceful restart;
  graceful restart time 0;
  long lived graceful restart yes;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd graceful;
}
protocol bgp node1 from nodes {
  neighbor 172.31.11.4 as 64512;
}
EOF
    docker exec bird-a1 birdcl configure

    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird/nodes-enetB.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export all;
  add paths on;
  graceful restart;
  graceful restart time 0;
  long lived graceful restart yes;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd graceful;
}
protocol bgp node1 from nodes {
  neighbor 172.31.21.4 as 64512;
}
EOF
    docker exec bird-b1 birdcl configure

    # Masquerade outbound traffic that is not from their own subnets.
    docker exec bird-a1 apk add --no-cache iptables
    docker exec bird-b1 apk add --no-cache iptables
    docker exec bird-a1 iptables -t nat -A POSTROUTING -o eth0 -d 172.31.0.0/16 -j ACCEPT
    docker exec bird-a1 iptables -t nat -A POSTROUTING -o eth0 -d 10.244.0.0/16 -j ACCEPT
    docker exec bird-a1 iptables -t nat -A POSTROUTING -o eth0 -d 10.96.0.0/16 -j ACCEPT
    docker exec bird-a1 iptables -t nat -A POSTROUTING -o eth0 ! -s 172.31.11.0/24 -j MASQUERADE
    docker exec bird-b1 iptables -t nat -A POSTROUTING -o eth0 -d 172.31.0.0/16 -j ACCEPT
    docker exec bird-b1 iptables -t nat -A POSTROUTING -o eth0 -d 10.244.0.0/16 -j ACCEPT
    docker exec bird-b1 iptables -t nat -A POSTROUTING -o eth0 -d 10.96.0.0/16 -j ACCEPT
    docker exec bird-b1 iptables -t nat -A POSTROUTING -o eth0 ! -s 172.31.21.0/24 -j MASQUERADE


    # Use kind to create and set up a 4 node Kubernetes cluster, with 2
    # nodes in rack A and 2 in rack B.
	  E_NETWORKS='[enetA, enetB, enetC]'
    I_NETWORK='[kind]'

    # ${KIND} load docker-image calico-test/busybox-with-reliable-nc

    # Fix rp_filter in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sysctl -w net.ipv4.conf.all.rp_filter=1

    # Fix /etc/resolv.conf in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sh -c "echo nameserver 8.8.8.8 > /etc/resolv.conf"

    # Create BGPConfiguration, BGPPeers.
    add_calico_resources

    exit 0

    install_tsee

    # Wait for installation to succeed and everything to be ready.
    for k8sapp in calico-node calico-kube-controllers calico-typha; do
	while ! time ${kubectl} wait pod --for=condition=Ready -l k8s-app=${k8sapp} -n calico-system --timeout=300s; do
	    # This happens when no matching resources exist yet,
	    # i.e. immediately after application of the Calico YAML.
	    sleep 5
	    ${kubectl} get po -A -o wide || true
	done
    done
    ${kubectl} get po -A -o wide

    # Edit the calico-node DaemonSet so we can make calico-node restarts take longer.
    ${KIND} get nodes | xargs -n1 -I {} kubectl label no {} ctd=f
    cat <<EOF | ${kubectl} patch ds calico-node -n calico-system --patch "$(cat -)"
metadata:
  annotations:
    unsupported.operator.tigera.io/ignore: "true"
spec:
  template:
    spec:
      nodeSelector:
        ctd: f
EOF

    # Check readiness again.
    for k8sapp in calico-node calico-kube-controllers calico-typha; do
	while ! time ${kubectl} wait pod --for=condition=Ready -l k8s-app=${k8sapp} -n calico-system --timeout=300s; do
	    # This happens when no matching resources exist yet,
	    # i.e. immediately after application of the Calico YAML.
	    sleep 5
	    ${kubectl} get po -A -o wide || true
	done
    done
    ${kubectl} get po -A -o wide

    # Show routing table everywhere.
    docker exec bird-a1 ip r
    docker exec bird-b1 ip r
    if ${DUAL}; then
	docker exec bird-a2 ip r
	docker exec bird-b2 ip r
    fi
    docker exec kind-control-plane ip r
    docker exec kind-worker ip r
    docker exec kind-worker2 ip r
    docker exec kind-worker3 ip r

    # Remove taints for master node, this would allow some test cases to run pod on master node.
    ${kubectl} taint node kind-control-plane node-role.kubernetes.io/master-
    ${kubectl} taint node kind-control-plane node-role.kubernetes.io/control-plane-

}

function do_cleanup {
    #${KIND} delete cluster || true
    #rm -f ${KIND}
    #docker rm -f `docker ps -a -q` || true
    docker rm -f bird-a1 bird-b1 bird-b2 bird-c1

    docker network disconnect enetA kind-worker
    docker network disconnect enetB kind-worker
    docker network disconnect enetC kind-worker
    docker network rm enetA enetB enetC || true

    docker network ls
    docker ps -a
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done

rm -rf ${tmpd}
