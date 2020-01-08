#!/bin/bash -ex

# gcr.io pull secrect credential file.
: ${GCR_IO_PULL_SECRET:=./docker_auth.json}

# kubectl binary.
: ${kubectl:=./kubectl}

# kind binary.
: ${KIND:=./tests/kind/bin/kind}

# Use locally built e2e.test.
E2E_TEST=~/go/src/github.com/tigera/k8s-e2e/bin/e2e.test

# Set config variables needed for ${kubectl} and calicoctl.
export KUBECONFIG=~/.kube/kind-config-kind
export DATASTORE_TYPE=kubernetes

# Normally, cleanup any leftover state, then setup, then test.
: ${STEPS:=cleanup setup}

# Set up second plane.
: ${DUAL:=true}

tmpd=$(mktemp -d -t calico.XXXXXX)

function load_image() {
    local node=$1
    docker cp ./cnx-node.tar ${node}:/cnx-node.tar
    docker exec -t ${node} ctr -n=k8s.io images import /cnx-node.tar
    docker exec -t ${node} rm /cnx-node.tar
}

function install_tsee() {

    manifest_base=tests/k8st/infra

    ${kubectl} apply -f ${manifest_base}/etcd.yaml

    cp ${manifest_base}/calico-etcd.yaml ${tmpd}/calico.yaml

    # Provide the cnx-node image to all the nodes and update the image in the daemonset.
    load_image kind-control-plane
    load_image kind-worker
    load_image kind-worker2
    load_image kind-worker3

    # Install pull secret so we can pull the right calicoctl.
    ${kubectl} -n kube-system create secret generic cnx-pull-secret \
	      --from-file=.dockerconfigjson=${GCR_IO_PULL_SECRET} \
	      --type=kubernetes.io/dockerconfigjson

    # Install using manifests which are Calico open source
    # Install Calico, with correct pod CIDR and without IP-in-IP.
    cat ${tmpd}/calico.yaml | \
        sed 's,192.168.0.0/16,10.244.0.0/16,' | \
        sed 's,"Always","Never",' | \
        sed 's,image: .*calico/node:.*,image: tigera/cnx-node:latest-amd64,' | \
        ${kubectl} apply -f -

    # Install Calicoctl on master node, avoid network disruption during bgp configuration.
    cat ${manifest_base}/calicoctl.yaml | \
        sed 's,hostNetwork: true,hostNetwork: true\n  nodeName: kind-control-plane,' | \
        ${kubectl} apply -f -
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
    docker network create --subnet=172.31.1.0/24 --ip-range=172.31.1.0/24 uplink
    docker network create --subnet=172.31.11.0/24 --ip-range=172.31.11.0/24 --gateway 172.31.11.2 ra1
    docker network create --subnet=172.31.21.0/24 --ip-range=172.31.21.0/24 --gateway 172.31.21.2 rb1

    # Create ToR routers.  (The 'neiljerram/birdy' image was built
    # from my locally modified projectcalico/bird repo; those changes
    # are still to be formalised and merged.)
    docker run -d --privileged --net=ra1 --ip=172.31.11.1 --name=bird-a1 neiljerram/birdy
    docker run -d --privileged --net=rb1 --ip=172.31.21.1 --name=bird-b1 neiljerram/birdy
    docker network connect --ip=172.31.1.2 uplink bird-a1
    docker network connect --ip=172.31.1.3 uplink bird-b1

    # Enable BFD function in the ToR routers.
    cat <<EOF | docker exec -i bird-a1 sed -i '2 r /dev/stdin' /etc/bird.conf
protocol bfd {
}
EOF
    cat <<EOF | docker exec -i bird-b1 sed -i '2 r /dev/stdin' /etc/bird.conf
protocol bfd {
}
EOF

    # Configure the ToR routers to peer with each other.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/peer-rb1.conf"
protocol bgp rb1 {
  description "Connection to BGP peer";
  local as 65001;
  gateway recursive;
  import all;
  export all;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  neighbor 172.31.1.3 as 65002;
  passive on;
  bfd on;
}
protocol static {
  route 172.31.10.0/23 via "eth0";
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
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  neighbor 172.31.1.2 as 65001;
  bfd on;
}
protocol static {
  route 172.31.20.0/23 via "eth0";
}
EOF
    docker exec bird-b1 birdcl configure

    if ${DUAL}; then
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
	docker run -d --privileged --net=ra2 --ip=172.31.12.1 --name=bird-a2 neiljerram/birdy
	docker run -d --privileged --net=rb2 --ip=172.31.22.1 --name=bird-b2 neiljerram/birdy
	docker network connect --ip=172.31.2.2 uplink2 bird-a2
	docker network connect --ip=172.31.2.3 uplink2 bird-b2
	cat <<EOF | docker exec -i bird-a2 sed -i '2 r /dev/stdin' /etc/bird.conf
protocol bfd {
}
EOF
	cat <<EOF | docker exec -i bird-b2 sed -i '2 r /dev/stdin' /etc/bird.conf
protocol bfd {
}
EOF
	cat <<EOF | docker exec -i bird-a2 sh -c "cat > /etc/bird/peer-rb2.conf"
protocol bgp rb2 {
  description "Connection to BGP peer";
  local as 65001;
  gateway recursive;
  import all;
  export all;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  neighbor 172.31.2.3 as 65002;
  passive on;
  bfd on;
}
protocol static {
  route 172.31.10.0/23 via "eth0";
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
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  neighbor 172.31.2.2 as 65001;
  bfd on;
}
protocol static {
  route 172.31.20.0/23 via "eth0";
}
EOF
	docker exec bird-b2 birdcl configure
    fi

    # Use kind to create and set up a 4 node Kubernetes cluster, with 2
    # nodes in rack A and 2 in rack B.
    if ${DUAL}; then
	RA_NETWORKS='[ra1, ra2]'
	RB_NETWORKS='[rb1, rb2]'
    else
	RA_NETWORKS='[ra1]'
	RB_NETWORKS='[rb1]'
    fi
    ${KIND} create cluster --config - <<EOF
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
  networks: ${RA_NETWORKS}
  loopback: 172.31.10.3
  routes:
  - "172.31.10.0/23 src 172.31.10.3 nexthop dev eth0 nexthop dev eth1"
  - "172.31.20.0/23 src 172.31.10.3 nexthop via 172.31.11.1 nexthop via 172.31.12.1"
- role: worker
  networks: ${RA_NETWORKS}
  loopback: 172.31.10.4
  routes:
  - "172.31.10.0/23 src 172.31.10.4 nexthop dev eth0 nexthop dev eth1"
  - "172.31.20.0/23 src 172.31.10.4 nexthop via 172.31.11.1 nexthop via 172.31.12.1"
- role: worker
  networks: ${RB_NETWORKS}
  loopback: 172.31.20.3
  routes:
  - "172.31.20.0/23 src 172.31.20.3 nexthop dev eth0 nexthop dev eth1"
  - "172.31.10.0/23 src 172.31.20.3 nexthop via 172.31.21.1 nexthop via 172.31.22.1"
- role: worker
  networks: ${RB_NETWORKS}
  loopback: 172.31.20.4
  routes:
  - "172.31.20.0/23 src 172.31.20.4 nexthop dev eth0 nexthop dev eth1"
  - "172.31.10.0/23 src 172.31.20.4 nexthop via 172.31.21.1 nexthop via 172.31.22.1"
EOF

    # Fix rp_filter in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sysctl -w net.ipv4.conf.all.rp_filter=1

    # Fix /etc/resolv.conf in each node.
    ${KIND} get nodes | xargs -n1 -I {} docker exec {} sh -c "echo nameserver 8.8.8.8 > /etc/resolv.conf"

    # On each node, change the interface-specific IPv4 addresses to
    # scope link, then recreate the default route (which was removed
    # by the address deletions and re-additions).
    #
    # Aside from the horrible quoting, and having it all on one line,
    # the code here is:
    #
    # for nic in eth0 eth1; do
    #     ip=`ip -4 a show dev $nic | grep inet | awk '{print $2;}'`
    #     ip a d $ip dev $nic
    #     ip a a $ip dev $nic scope link
    # done
    # ip r a default via ${ip%.*}.2 src ${ip%/*}
    #
    for n in kind-control-plane kind-worker kind-worker2 kind-worker3; do
	docker exec $n /bin/bash -x -c "for nic in eth0 eth1; do ip=\`ip -4 a show dev \$nic | grep inet | awk '{print \$2;}'\`; ip a d \$ip dev \$nic; ip a a \$ip dev \$nic scope link; done; ip r a default via \${ip%.*}.2 src \${ip%/*}"
    done

    install_tsee

    # Wait for calicoctl to be ready.
    while ! time ${kubectl} wait pod calicoctl --for=condition=Ready -n kube-system --timeout=300s; do
        # This happens when no matching resources exist yet,
        # i.e. immediately after application of the Calico YAML.
        sleep 5
    done
    ${kubectl} get po --all-namespaces -o wide

    echo "sleep 30 seconds"
    sleep 30

    # Disable the full mesh and configure Calico nodes to peer instead
    # with their ToR.
    cat << EOF >> ${tmpd}/bgp.yaml
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
EOF

    ${kubectl} cp ${tmpd}/bgp.yaml calicoctl:/tmp/bgp.yaml -n kube-system
    ${kubectl} exec -t -n kube-system calicoctl -- /calicoctl apply -f /tmp/bgp.yaml
    rm ${tmpd}/bgp.yaml

    # Correspondingly, configure the ToR end of those peerings.
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/nodes-ra1.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 65001;
  direct;
  gateway recursive;
  import all;
  export all;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd on;
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
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd on;
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

    if ${DUAL}; then
        echo "song DUAL setup"
	# Similar peering config for the second plane.
        cat << EOF >> ${tmpd}/bgp.yaml
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
        ${kubectl} cp ${tmpd}/bgp.yaml calicoctl:/tmp/bgp.yaml -n kube-system
        ${kubectl} exec -t -n kube-system calicoctl -- /calicoctl apply -f /tmp/bgp.yaml
        rm ${tmpd}/bgp.yaml
        echo "song DUAL setup done"

	cat <<EOF | docker exec -i bird-a2 sh -c "cat > /etc/bird/nodes-ra2.conf"
template bgp nodes2 {
  description "Connection to BGP peer";
  local as 65001;
  direct;
  gateway recursive;
  import all;
  export all;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd on;
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
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
  bfd on;
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
    fi

    # Wait a few seconds, then check routing table everywhere.
    sleep 5
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

    # Setup ippool for loopback addresses.
    cat << EOF >> ${tmpd}/ippool.yaml
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
EOF

    ${kubectl} cp ${tmpd}/ippool.yaml calicoctl:/tmp/ippool.yaml -n kube-system
    ${kubectl} exec -t -n kube-system calicoctl -- /calicoctl apply -f /tmp/ippool.yaml
    rm ${tmpd}/ippool.yaml

    # For the nodes in rack A...
    for n in kind-control-plane kind-worker; do
	# Configure AS number 65001.
	# Label as being in rack A.
	${kubectl} label no $n rack=ra

        ${kubectl} exec -t calicoctl -n kube-system -- /calicoctl patch node $n --patch '{"spec":{"bgp": {"asNumber": "65001"}}}'
    done

    # Similarly, but AS number 65002, for the nodes in rack B.
    for n in kind-worker2 kind-worker3; do
	${kubectl} label no $n rack=rb
        ${kubectl} exec -t calicoctl -n kube-system -- /calicoctl patch node $n --patch '{"spec":{"bgp": {"asNumber": "65002"}}}'
    done

    # Wait for installation to succeed and everything to be ready.
    for k8sapp in calico-node kube-dns calico-kube-controllers; do
	while ! time ${kubectl} wait pod --for=condition=Ready -l k8s-app=${k8sapp} -n kube-system --timeout=300s; do
	    # This happens when no matching resources exist yet,
	    # i.e. immediately after application of the Calico YAML.
	    sleep 5
	done
    done
    ${kubectl} get po --all-namespaces -o wide

    # Remove taints for master node, this would allow some test cases to run pod on master node.
    ${kubectl} taint node kind-control-plane node-role.kubernetes.io/master-

}

function break_plane {
    plane=$1
    docker exec bird-a${plane} ip link set dev eth1 down
}

function restore_plane {
    plane=$1
    docker exec bird-a${plane} ip link set dev eth1 up
}

function do_resilience {
    # Create test client on kind-worker (rack A).
    ${kubectl} apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  labels:
    pod-name: client
  name: client
spec:
  containers:
  - args:
    - /bin/sh
    - -c
    - sleep 360000
    image: busybox
    imagePullPolicy: Always
    name: client
  nodeSelector:
    beta.kubernetes.io/os: linux
  nodeName: kind-worker
EOF

    # Create test server on kind-worker3 (rack B).
    ${kubectl} apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  labels:
    pod-name: server
  name: server
spec:
  containers:
  - args:
    - /bin/sh
    - -c
    - sleep 360000
    image: busybox
    imagePullPolicy: Always
    name: server
  nodeSelector:
    beta.kubernetes.io/os: linux
  nodeName: kind-worker3
EOF

    # Start server listening.
    ${kubectl} exec server -- nc -l -p 8080 > rcvd.txt &

    # Get server IP.
    server_ip=`${kubectl} get po server -o wide | tail -1 | awk '{print $6;}'`

    # Send from client.
    time ${kubectl} exec client -- sh -c 'for i in `seq 1 3000`; do echo "$i -- dual tor test"; sleep 0.01; done |'" nc -w 1 ${server_ip} 8080" &

    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt

    break_plane 1

    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt

    restore_plane 1

    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt

    break_plane 1

    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt
    sleep 5
    wc -c rcvd.txt

    restore_plane 1

    wait
}

function do_test {
    time ${E2E_TEST} --kubeconfig=$KUBECONFIG -ginkgo.focus="NetworkPolicy and GlobalNetworkPolicy"
}

function do_cleanup {
    ${KIND} delete cluster
    docker rm -f `docker ps -a -q` || true
    docker network rm ra2 rb2 uplink2 || true
    docker network rm ra1 rb1 uplink || true
    docker network ls
    docker ps -a
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done
