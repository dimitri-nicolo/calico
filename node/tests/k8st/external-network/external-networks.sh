#!/bin/bash -ex

# test directory.
TEST_DIR=./tests/k8st

# kubectl binary.
: ${kubectl:=./kubectl}

# kind binary.
: ${KIND:=dist/kind}

echo "Download kind executable with multiple networks support"
curl -L https://github.com/projectcalico/kind/releases/download/multiple-networks-0.3/kind -o ${KIND}
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
kind: BGPConfiguration
metadata:
  name: default
spec:
  nodeToNodeMeshEnabled: false
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: enetA
spec:
  nodeSelector: "rack == 'ra'"
  peerIP: 172.31.11.1
  asNumber: 65001
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: enetB
spec:
  nodeSelector: "rack == 'rb'"
  peerIP: 172.31.21.1
  asNumber: 65002
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
---
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: ra.loopback
spec:
  cidr: 172.31.10.0/24
  disabled: true
  nodeSelector: all()
---
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: rb.loopback
spec:
  cidr: 172.31.20.0/24
  disabled: true
  nodeSelector: all()
---
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: rb.loopback
spec:
  cidr: 172.31.20.0/24
  disabled: true
  nodeSelector: all()
---
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: default-ipv4
spec:
  cidr: 10.244.0.0/16
  nodeSelector: all()
EOF

    if ${DUAL}; then
	# Add BGP peering config for the second plane.
	${CALICOCTL} apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: ra2
spec:
  nodeSelector: "rack == 'ra'"
  peerIP: 172.31.12.1
  asNumber: 65001
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: rb2
spec:
  nodeSelector: "rack == 'rb'"
  peerIP: 172.31.22.1
  asNumber: 65002
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
EOF
    fi
}

function install_tsee() {

    # Load the locally built cnx-node image into the KIND cluster.
    ${KIND} load docker-image tigera/cnx-node:latest-amd64

    # Inside the cluster, retag so that it appears to be the image that the operator will
    # look for.
    for node in kind-control-plane kind-worker kind-worker2 kind-worker3; do
	docker exec $node ctr --namespace=k8s.io images tag docker.io/tigera/cnx-node:latest-amd64 ${CNX_NODE_IMAGE}
	docker exec $node crictl images
    done

    # Prepare for an operator install.
    ${kubectl} create -f ${OPERATOR_URL}

    # Install pull secret.
    ${kubectl} create secret generic tigera-pull-secret \
	      --from-file=.dockerconfigjson=${GCR_IO_PULL_SECRET} \
	      --type=kubernetes.io/dockerconfigjson -n tigera-operator

    # Create BGPConfiguration, BGPPeers and IPPools.
    add_calico_resources

    # Label and annotate nodes.
    ${kubectl} label node kind-control-plane rack=ra
    ${kubectl} label node kind-worker rack=ra
    ${kubectl} label node kind-worker2 rack=rb
    ${kubectl} label node kind-worker3 rack=rb
    ${kubectl} annotate node kind-control-plane projectcalico.org/ASNumber=65001
    ${kubectl} annotate node kind-worker projectcalico.org/ASNumber=65001
    ${kubectl} annotate node kind-worker2 projectcalico.org/ASNumber=65002
    ${kubectl} annotate node kind-worker3 projectcalico.org/ASNumber=65002

    # Create Installation resource to kick off the install.
    ${kubectl} apply -f - <<EOF
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  variant: TigeraSecureEnterprise
  imagePullSecrets:
    - name: tigera-pull-secret
EOF
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
    docker network create --subnet=172.18.0.0/24 --ip-range=172.18.0.0/24 --gateway 172.18.0.2 internal
    docker network create --subnet=172.31.11.0/24 --ip-range=172.31.11.0/24 --gateway 172.31.11.2 enetA
    docker network create --subnet=172.31.21.0/24 --ip-range=172.31.21.0/24 --gateway 172.31.21.2 enetB
    docker network create --subnet=172.31.31.0/24 --ip-range=172.31.31.0/24 --gateway 172.31.31.2 enetC

    # Create routers for external networks.
    docker run -d --privileged --net=enetA --ip=172.31.11.1 --name=bird-a1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.1 --name=bird-b1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.2 --name=bird-b2 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetC --ip=172.31.31.1 --name=bird-c1 ${ROUTER_IMAGE}

    # Configure graceful restart.
    make_bird_graceful bird-a1
    make_bird_graceful bird-b1
    make_bird_graceful bird-b2
    make_bird_graceful bird-c1

    # Configure Router end of cluster node peerings.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/nodes-enetA.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 65001;
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
  neighbor 172.31.11.3 as 65001;
  rr client;
}
protocol bgp node2 from nodes {
  neighbor 172.31.11.4 as 65001;
  rr client;
}
EOF
    docker exec bird-a1 birdcl configure
    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird/nodes-enetB.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 65002;
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
  neighbor 172.31.21.3 as 65002;
  rr client;
}
protocol bgp node2 from nodes {
  neighbor 172.31.21.4 as 65002;
  rr client;
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
    I_NETWORK='[internal]'

    ${KIND} create cluster --image kindest/node:$(K8S_VERSION) --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
  networks: ${I_NETWORK}
- role: worker
  networks: ${E_NETWORKS}
- role: worker
  networks: ${E_NETWORKS}
- role: worker
  networks: ${I_NETWORK}
kubeadmConfigPatches:
- |
  apiVersion: kubeproxy.config.k8s.io/v1alpha1
  kind: KubeProxyConfiguration
  metadata:
    name: config
  conntrack:
    maxPerCore: 0

EOF

    ${KIND} load docker-image calico-test/busybox-with-reliable-nc

    # Fix rp_filter in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sysctl -w net.ipv4.conf.all.rp_filter=1

    # Fix /etc/resolv.conf in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sh -c "echo nameserver 8.8.8.8 > /etc/resolv.conf"

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
    ${KIND} delete cluster || true
    rm -f ${KIND}
    docker rm -f `docker ps -a -q` || true
    docker network rm enetA enetB enetC internal || true
    docker network ls
    docker ps -a
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done

rm -rf ${tmpd}
