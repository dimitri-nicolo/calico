#!/bin/bash -ex

# test directory.
TEST_DIR=./tests/k8st

# kubectl binary.
: ${kubectl:=../hack/test/kind/kubectl}

# kind binary.
: ${KIND:=dist/kind}

echo "Download kind executable with multiple networks support"
mkdir -p $(dirname ${KIND})
curl -L https://github.com/projectcalico/kind/releases/download/multiple-networks-0.3/kind -o ${KIND}
chmod +x ${KIND}

# Set config variables needed for ${kubectl} and calicoctl.
export KUBECONFIG=~/.kube/kind-config-kind

# Normally, cleanup any leftover state, then setup, then test.
: ${STEPS:=cleanup setup}

# Changes the default routes for a single kind worker, and connects it to an additional network (one which is not related to dual-tor).
: ${LEGACY_ROUTING:=false}

# URL for an operator install. We will build the local code and load the images so
# that the operator will use thoso images with local changes instead of pulling
# images.
: ${OPERATOR_CRDS_URL:=../manifests/operator-crds.yaml}
: ${OPERATOR_URL:=../manifests/tigera-operator.yaml}

tmpd=$(mktemp -d -t calico.XXXXXX)

function make_bird_graceful() {
    node=$1
    docker exec -i $node sed -i '/protocol kernel {/r /dev/stdin' /etc/bird.conf <<EOF
    persist;          # Don't remove routes on bird shutdown
    graceful restart; # Turn on graceful restart to reduce potential flaps in
                      # routes when reloading BIRD configuration.  With a full
                      # automatic mesh, there is no way to prevent BGP from
                      # flapping since multiple nodes update their BGP
                      # configuration at the same time, GR is not guaranteed to
                      # work correctly in this scenario.
EOF
}

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
  name: ra1
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
  name: rb1
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
}

function add_bgp_layout() {
    DUAL_TOR_DIR=tests/k8st/dual-tor
    CFG_FILE=$1
    cat >${DUAL_TOR_DIR}/bgp-layout.yaml <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: bgp-layout
  namespace: tigera-operator
data:
  earlyNetworkConfiguration: |
EOF
    sed -e 's/^/    /' ${DUAL_TOR_DIR}/${CFG_FILE} >>${DUAL_TOR_DIR}/bgp-layout.yaml
    ${kubectl} create -f ${DUAL_TOR_DIR}/bgp-layout.yaml
}

function install_tsee() {

    # Load the locally built images into the KIND cluster nodes.
    . ${TEST_DIR}/load_images_on_kind_cluster.sh

    # Prepare for an operator install.
    ${kubectl} create -f ${OPERATOR_CRDS_URL}
    ${kubectl} create -f ${OPERATOR_URL}

    # Install pull secret.
    ${kubectl} create secret generic tigera-pull-secret \
	      --from-file=.dockerconfigjson=${GCR_IO_PULL_SECRET} \
	      --type=kubernetes.io/dockerconfigjson -n tigera-operator

    # Create BGPConfiguration, BGPPeers and IPPools.
    add_calico_resources

    ENC_FILE=$1
    # Create bgp-layout ConfigMap.
    add_bgp_layout ${ENC_FILE}

    # Label and annotate nodes.
    ${kubectl} label node kind-control-plane rack=ra
    ${kubectl} label node kind-worker rack=ra
    ${kubectl} label node kind-worker2 rack=rb
    ${kubectl} label node kind-worker3 rack=rb

    if ${LEGACY_ROUTING}; then
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

    else
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
  calicoNetwork:
    nodeAddressAutodetectionV4:
      skipInterface: eth2
EOF
    fi
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
    #           |  'ra1'                        |  'rb1'
    #           |                               |
    #           |.3 .4                          |.3 .4
    #  +-----------------+             +-----------------+
    #  | Nodes of rack A |             | Nodes of rack B |
    #  +-----------------+             +-----------------+
    #     kind-control-plane              kind-worker2
    #     kind-worker                     kind-worker3
    #
    #
    # Note: when not in legacy mode, 'kind-worker' will also be connected to another network 'rc'.
    # 'rc' will have a dedicated, and isolated router 'bird-c', which will
    # provide the default route / internet uplink for 'kind-worker'.
    # Calico should be able to provide dual-tor capabilities for 'kind-worker' despite this.
    docker network create --subnet=172.31.1.0/24 --ip-range=172.31.1.0/24 uplink
    docker network create --subnet=172.31.11.0/24 --ip-range=172.31.11.0/24 --gateway 172.31.11.2 ra1
    docker network create --subnet=172.31.21.0/24 --ip-range=172.31.21.0/24 --gateway 172.31.21.2 rb1

    if ! ${LEGACY_ROUTING}; then
      docker network create --subnet=172.31.31.0/24 --ip-range=172.31.31.0/24 --gateway 172.31.31.2 rc
    fi

    # Create ToR routers.
    docker run -d --privileged --net=ra1 --ip=172.31.11.1 --name=bird-a1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=rb1 --ip=172.31.21.1 --name=bird-b1 ${ROUTER_IMAGE}
    docker network connect --ip=172.31.1.2 uplink bird-a1
    docker network connect --ip=172.31.1.3 uplink bird-b1

    if ! ${LEGACY_ROUTING}; then
      docker run -d --privileged --net=rc --ip=172.31.31.1 --name=bird-c ${ROUTER_IMAGE}
    fi

    # Configure graceful restart.
    make_bird_graceful bird-a1
    make_bird_graceful bird-b1
    if ! ${LEGACY_ROUTING}; then
      make_bird_graceful bird-c
    fi

    # Configure the ToR routers to peer with each other.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/peer-rb1.conf"
protocol bgp rb1 {
  description "Connection to BGP peer";
  local as 65001;
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
  neighbor 172.31.1.3 as 65002;
  passive on;
  bfd graceful;
}
EOF
    docker exec bird-a1 birdcl configure
    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird/peer-ra1.conf"
protocol bgp ra1 {
  description "Connection to BGP peer";
  local as 65002;
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
  neighbor 172.31.1.2 as 65001;
  bfd graceful;
}
EOF
    docker exec bird-b1 birdcl configure

    # Configure ToR end of cluster node peerings.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/nodes-ra1.conf"
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
    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird/nodes-rb1.conf"
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

    if ! ${LEGACY_ROUTING}; then
      cat <<EOF | docker exec -i bird-c sh -c "cat > /etc/bird/nodes-rc.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 65003;
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
  neighbor 172.31.31.3 as 65002;
  rr client;
  export filter {
    if net = 0.0.0.0/0 then accept;
  }
}

EOF
      docker exec bird-c birdcl configure
    fi

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

    if ! ${LEGACY_ROUTING}; then
      docker exec bird-c apk add --no-cache iptables
      docker exec bird-c iptables -t nat -A POSTROUTING -o eth0 -d 172.31.0.0/16 -j ACCEPT
      docker exec bird-c iptables -t nat -A POSTROUTING -o eth0 -d 10.244.0.0/16 -j ACCEPT
      docker exec bird-c iptables -t nat -A POSTROUTING -o eth0 -d 10.96.0.0/16 -j ACCEPT
      docker exec bird-c iptables -t nat -A POSTROUTING -o eth0 ! -s 172.31.31.0/24 -j MASQUERADE
      # The box fails to forward internet-destined traffic from workers without this line.
      docker exec bird-c iptables -t nat -A POSTROUTING -o eth0 ! -d 172.31.0.0/16 -j MASQUERADE
    fi


	# Now with a second connectivity plane, that becomes:
	#
	#   +---------------+    172.31.2   +---------------+
	#   | ToR (bird-a2) |---------------| ToR (bird-b2) |
	#   +---------------+ .2         .3 +---------------+
	#     |                                |
	#     |  +---------------+    172.31.1 | +---------------+
	#     |  | ToR (bird-a1) |---------------| ToR (bird-b1) |
	#     |  +---------------+ .2         .3 +---------------+
	#     |         |                      |        |
	#     |         |                      |        |
	#  172.13.12    |                   172.13.22   |
	#     |         | 172.31.11            |        | 172.31.21
	#     |         |                      |        |
	#     |         |                      |        |
	#     |         |                      |        |
	#   +-----------------+             +-----------------+
	#   | Nodes of rack A |             | Nodes of rack B |
	#   +-----------------+             +-----------------+
	#      kind-control-plane              kind-worker2
	#      kind-worker                     kind-worker3
	docker network create --subnet=172.31.2.0/24 --ip-range=172.31.2.0/24 uplink2
	docker network create --subnet=172.31.12.0/24 --ip-range=172.31.12.0/24 --gateway 172.31.12.2 ra2
	docker network create --subnet=172.31.22.0/24 --ip-range=172.31.22.0/24 --gateway 172.31.22.2 rb2
	docker run -d --privileged --net=ra2 --ip=172.31.12.1 --name=bird-a2 ${ROUTER_IMAGE}
	docker run -d --privileged --net=rb2 --ip=172.31.22.1 --name=bird-b2 ${ROUTER_IMAGE}
	docker network connect --ip=172.31.2.2 uplink2 bird-a2
	docker network connect --ip=172.31.2.3 uplink2 bird-b2
	make_bird_graceful bird-a2
	make_bird_graceful bird-b2
	cat <<EOF | docker exec -i bird-a2 sh -c "cat > /etc/bird/peer-rb2.conf"
protocol bgp rb2 {
  description "Connection to BGP peer";
  local as 65001;
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
  neighbor 172.31.2.3 as 65002;
  passive on;
  bfd graceful;
}
EOF
	docker exec bird-a2 birdcl configure
	cat <<EOF | docker exec -i bird-b2 sh -c "cat > /etc/bird/peer-ra2.conf"
protocol bgp ra2 {
  description "Connection to BGP peer";
  local as 65002;
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
  neighbor 172.31.2.2 as 65001;
  bfd graceful;
}
EOF
	docker exec bird-b2 birdcl configure

	# Configure ToR end of cluster node peerings.
	cat <<EOF | docker exec -i bird-a2 sh -c "cat > /etc/bird/nodes-ra2.conf"
template bgp nodes2 {
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
protocol bgp node1 from nodes2 {
  neighbor 172.31.12.3 as 65001;
  rr client;
}
protocol bgp node2 from nodes2 {
  neighbor 172.31.12.4 as 65001;
  rr client;
}
EOF
	docker exec bird-a2 birdcl configure
	cat <<EOF | docker exec -i bird-b2 sh -c "cat > /etc/bird/nodes-rb2.conf"
template bgp nodes2 {
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
protocol bgp node1 from nodes2 {
  neighbor 172.31.22.3 as 65002;
  rr client;
}
protocol bgp node2 from nodes2 {
  neighbor 172.31.22.4 as 65002;
  rr client;
}
EOF
	docker exec bird-b2 birdcl configure

	# Masquerade outbound traffic that is not from their own subnets.
	docker exec bird-a2 apk add --no-cache iptables
	docker exec bird-b2 apk add --no-cache iptables
	docker exec bird-a2 iptables -t nat -A POSTROUTING -o eth0 -d 172.31.0.0/16 -j ACCEPT
	docker exec bird-a2 iptables -t nat -A POSTROUTING -o eth0 -d 10.244.0.0/16 -j ACCEPT
	docker exec bird-a2 iptables -t nat -A POSTROUTING -o eth0 -d 10.96.0.0/16 -j ACCEPT
	docker exec bird-a2 iptables -t nat -A POSTROUTING -o eth0 ! -s 172.31.12.0/24 -j MASQUERADE
	docker exec bird-b2 iptables -t nat -A POSTROUTING -o eth0 -d 172.31.0.0/16 -j ACCEPT
	docker exec bird-b2 iptables -t nat -A POSTROUTING -o eth0 -d 10.244.0.0/16 -j ACCEPT
	docker exec bird-b2 iptables -t nat -A POSTROUTING -o eth0 -d 10.96.0.0/16 -j ACCEPT
	docker exec bird-b2 iptables -t nat -A POSTROUTING -o eth0 ! -s 172.31.22.0/24 -j MASQUERADE

    # Use kind to create and set up a 4 node Kubernetes cluster, with 2
    # nodes in rack A and 2 in rack B.
      RA_NETWORKS='[ra1, ra2]'
      if ! ${LEGACY_ROUTING}; then
        WORKER_RA_NETWORKS='[ra1, ra2, rc]'
      else
        WORKER_RA_NETWORKS='[ra1, ra2]'
      fi
      RB_NETWORKS='[rb1, rb2]'


    ${KIND} create cluster --image calico/dual-tor-node --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
  networks: ${RA_NETWORKS}
- role: worker
  networks: ${WORKER_RA_NETWORKS}
- role: worker
  networks: ${RB_NETWORKS}
- role: worker
  networks: ${RB_NETWORKS}
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

    for node in kind-control-plane kind-worker kind-worker2 kind-worker3; do
      echo ===== docker exec $node ip r
      docker exec $node ip r
    done
    echo ===== ${kubectl} get nodes -o yaml
    ${kubectl} get nodes -o yaml

    if ! ${LEGACY_ROUTING}; then
      # make kind-worker's default route go over network 'rc' via bird-c.
      # Calico EN should respect this default route and avoid clobbering it with dual-tor routes.
      docker exec kind-worker ip route replace default via 172.31.31.1
      docker exec kind-worker podman restart calico-early
    fi

    if ! ${LEGACY_ROUTING}; then
      install_tsee cfg.yaml
    else
      install_tsee cfg-legacy.yaml
    fi

    # Wait for installation to succeed and everything to be ready.
    for k8sapp in calico-node calico-kube-controllers calico-typha; do
      num_failed_waits=0
      while ! time ${kubectl} wait pod --for=condition=Ready -l k8s-app=${k8sapp} -n calico-system --timeout=300s; do
        # This happens when no matching resources exist yet,
        # i.e. immediately after application of the Calico YAML.
        sleep 5
        ${kubectl} get po -A -o wide || true
        let 'num_failed_waits++' || true
        if [ $num_failed_waits -gt 10 ]; then
          echo ERROR: Timed out waiting for ${k8sapp}
          for node in kind-control-plane kind-worker kind-worker2 kind-worker3; do
            echo ===== docker exec $node ip r
            docker exec $node ip r
          done
          echo ===== ${kubectl} get ds -A -o yaml
          ${kubectl} get ds -A -o yaml
          echo ===== ${kubectl} get nodes -o yaml
          ${kubectl} get nodes -o yaml
          echo ===== ${kubectl} get svc -A -o yaml
          ${kubectl} get svc -A -o yaml
          echo ===== ${kubectl} get ep -A -o yaml
          ${kubectl} get ep -A -o yaml
          for p in `${kubectl} get po -n calico-system | awk '{print $1;}' | grep calico-node-`; do
            echo ===== ${kubectl} logs $p -n calico-system -c calico-node
            ${kubectl} logs $p -n calico-system -c calico-node || true
            echo ===== ${kubectl} logs $p -n calico-system -c calico-node --previous
            ${kubectl} logs $p -n calico-system -c calico-node --previous || true
          done
          for k8sapp in calico-typha; do
            echo ===== ${kubectl} logs -l k8s-app=${k8sapp} -n calico-system --all-containers --ignore-errors --prefix --max-log-requests 42 --tail 30000
            ${kubectl} logs -l k8s-app=${k8sapp} -n calico-system --all-containers --ignore-errors --prefix --max-log-requests 42 --tail 30000
          done
          echo ===== ${kubectl} logs -l k8s-app=calico-kube-controllers -n calico-system --all-containers --ignore-errors --prefix --max-log-requests 42 --tail 30000
          ${kubectl} logs -l k8s-app=calico-kube-controllers -n calico-system --all-containers --ignore-errors --prefix --max-log-requests 42 --tail 30000
          exit 1
        fi
      done
    done
    ${kubectl} get po -A -o wide

    # Edit the calico-node DaemonSet so we can make calico-node restarts take longer.
    ${KIND} get nodes | xargs -n1 -I {} ${kubectl} label no {} ctd=f
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
    if ! ${LEGACY_ROUTING}; then
      docker exec bird-c ip r
    fi

    docker exec bird-a2 ip r
    docker exec bird-b2 ip r
    docker exec kind-control-plane ip r
    docker exec kind-worker ip r
    docker exec kind-worker2 ip r
    docker exec kind-worker3 ip r

    # Remove taints for master node, this would allow some test cases to run pod on master node.
    ${kubectl} taint node kind-control-plane node-role.kubernetes.io/master-
    ${kubectl} taint node kind-control-plane node-role.kubernetes.io/control-plane-

    if ! ${LEGACY_ROUTING}; then
      if ! validate_routing; then
        echo "Couldn't validate routing. Dumping kind node network info..."
        for n in kind-control-plane kind-worker kind-worker1 kind-worker2; do
          docker exec $n ip route
          docker exec $n ip addr
        done
        exit 1
      fi
    fi
}

function validate_routing {
  kind_worker_default_route=$(docker exec kind-worker ip route show default)
  # Expect 'default via 172.31.31.1'.
  # Kind-worker started with no default routing to the ToR networks,
  # and a default route to 'rc', which is not mentioned in the early-network config.
  # Calico-early should preserve this configuration.
  if [[ ! ${kind_worker_default_route} =~ "172.31.31.1" ]]; then
    echo "Kind worker 0 has invalid default route. Expected a route to 172.31.31.1"
    return 1
  fi

  kind_worker_num_default_routes=$(echo "${kind_worker_default_route}" | wc -l)
  if [[ ${kind_worker_num_default_routes} -ne 1 ]]; then
    echo "Kind worker 0 has an unexpected number of routes. Expected one."
    return 1
  fi

  # For nodes which have a default route for both ToRs, we expect default routing output to look like:
  # default proto bird
  #   nexthop via 172.31.11.1 dev eth0 weight 1
  #   nexthop via 172.31.12.1 dev eth1 weight 1
  kind_control_plane_default_route=$(docker exec kind-control-plane ip route show default)
  if [[ ! ${kind_control_plane_default_route} =~ "172.31.11.1" ]]; then
    echo "Kind control-plane node has invalid default routing. Expected a route to 172.31.11.1."
    return 1
  fi
  if [[ ! ${kind_control_plane_default_route} =~ "172.31.12.1" ]]; then
    echo "Kind control-plane node has invalid default routing. Expected a route to 172.31.12.1"
    return 1
  fi
  kind_control_plane_num_default_route_lines=$(echo "${kind_control_plane_default_route}" | wc -l)
  if [[ ${kind_control_plane_num_default_route_lines} -ne 3 ]]; then
    echo "Kind control-plane did not have the expected number of default routes. Expected 3 lines of output from `ip route show default`."
    return 1
  fi

  # For kind-workers 2 and 3, we expect default routing to look like:
  # wrkr_default_routes='default proto bird
  #   nexthop via 172.31.21.1 dev eth0 weight 1
  #   nexthop via 172.31.22.1 dev eth1 weight 1
  for wrkr in kind-worker2 kind-worker3; do
    wrkr_default_routes=$(docker exec ${wrkr} ip route show default)
    if [[ ! ${wrkr_default_routes} =~ "172.31.21.1" ]]; then
      echo "${wrkr} did not have the expected default routing. Missing a route to 172.31.21.1."
      return 1
    fi
    if [[ ! ${wrkr_default_routes} =~ "172.31.22.1" ]]; then
      echo "${wrkr} did not have the expected default routing. Missing a route to 172.31.22.1."
      return 1
    fi

    # Route output should have a root "default proto bird" line, followed by two nexthops.
    wrkr_default_routes_num_lines=$(echo "${wrkr_default_routes}" | wc -l)
    if [[ ${wrkr_default_routes_num_lines} -ne 3 ]]; then
      echo "${wrkr} did not have the expected number of default routes. Expected 3 lines of output from 'ip route show default'."
      return 1
    fi
  done
}

function do_cleanup {
    ${KIND} delete cluster || true
    rm -f ${KIND}
    docker rm -f `docker ps -a -q` || true
    docker network rm ra2 rb2 uplink2 || true
    docker network rm ra1 rb1 uplink || true
    docker network ls
    docker ps -a

    rm ${DUAL_TOR_DIR}/bgp-layout.yaml || true
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done

rm -rf ${tmpd}
