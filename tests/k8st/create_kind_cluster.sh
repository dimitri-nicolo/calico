#!/bin/bash -e

# test directory.
TEST_DIR=./tests/k8st

# gcr.io pull secrect credential file.
: ${GCR_IO_PULL_SECRET:=./docker_auth.json}

# Path to Enteprise product license
: ${TSEE_TEST_LICENSE:=/home/semaphore/secrets/new-test-customer-license.yaml}

# kubectl binary.
: ${kubectl:=./kubectl}

# kind binary.
: ${KIND:=$TEST_DIR/kind}

# type of rig to set up
: ${K8ST_RIG:=dual_stack}

# Set config variables needed for ${kubectl}.
export KUBECONFIG=~/.kube/kind-config-kind

function dual_stack {
    test "${K8ST_RIG}" = dual_stack
}

function checkModule(){
  MODULE="$1"
  echo "Checking kernel module $MODULE ..."
  if lsmod | grep "$MODULE" &> /dev/null ; then
    return 0
  else
    return 1
  fi
}

function load_image() {
    local node=$1
    docker cp ./cnx-node.tar ${node}:/cnx-node.tar
    docker exec -t ${node} ctr -n=k8s.io images import /cnx-node.tar
    docker exec -t ${node} rm /cnx-node.tar
}

function update_calico_manifest() {
    local yaml=$1
    if dual_stack; then
	# Based on instructions in http://docs.projectcalico.org/master/networking/dual-stack.md
	# add assign_ipv4 and assign_ipv6 to CNI config
	sed -i -e '/"type": "calico-ipam"/r /dev/stdin' "${yaml}" <<EOF
              "assign_ipv4": "true",
              "assign_ipv6": "true"
EOF
	sed -i -e 's/"type": "calico-ipam"/"type": "calico-ipam",/' "${yaml}"

	sed -i -e '/"type": "calico"/r /dev/stdin' "${yaml}" <<EOF
     "feature_control": {
         "floating_ips": true
     },
EOF

	# And add all the IPV6 env vars
	sed -i '/# Enable IPIP/r /dev/stdin' "${yaml}" << EOF
            - name: IP6
              value: "autodetect"
            - name: CALICO_IPV6POOL_CIDR
              value: "fd00:10:244::/64"
EOF
	# update FELIX_IPV6SUPPORT=true
	sed -i '/FELIX_IPV6SUPPORT/!b;n;c\              value: "true"' "${yaml}"
    else
	# For vanilla setup, we don't want any IP-IP or VXLAN overlay.
	sed -i 's/Always/Never/' "${yaml}"
    fi

    # update calico/node image
    sed -i 's,image: .*calico/node:.*,image: tigera/cnx-node:latest-amd64,' "${yaml}"
}

if dual_stack; then
    echo "kubernetes dualstack requires ipvs mode kube-proxy for the moment."
    MODULES=("ip_vs" "ip_vs_rr" "ip_vs_wrr" "ip_vs_sh" "nf_conntrack_ipv4")
    for m in "${MODULES[@]}"; do
	checkModule $m || {
	    echo "Could not find kernel module $m. install it..."
	    # Modules could be built into kernel and not exist as a kernel module anymore.
	    # For instance, kernel 5.0.0 ubuntu has nf_conntrack_ipv4 built in.
	    # So try to install modules required and continue if it failed..
	    sudo modprobe $m || true
	}
    done
    echo
fi

if dual_stack; then
    echo "Download kind executable with dual stack support"
    # We need to replace kind executable and node image
    # with official release once dual stack is fully supported by upstream.
    curl -L https://github.com/song-jiang/kind/releases/download/dualstack-1.17.0/kind -o ${KIND}
else
    echo "Download latest upstream kind executable"
    curl -L https://github.com/kubernetes-sigs/kind/releases/download/v0.8.1/kind-linux-amd64 -o ${KIND}
fi
chmod +x ${KIND}

echo "Create kind cluster"
set +e
for attempt in 1 2 3; do
    if dual_stack; then
	${KIND} -v 1 create cluster --image songtjiang/kindnode-dualstack:1.17.0 --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  podSubnet: "192.168.128.0/17,fd00:10:244::/64"
  ipFamily: DualStack
nodes:
# the control plane node
- role: control-plane
- role: worker
- role: worker
- role: worker
kubeadmConfigPatches:
- |
  apiVersion: kubeadm.k8s.io/v1beta2
  kind: ClusterConfiguration
  metadata:
    name: config
  featureGates:
    IPv6DualStack: true
- |
  apiVersion: kubeproxy.config.k8s.io/v1alpha1
  kind: KubeProxyConfiguration
  metadata:
    name: config
  mode: ipvs
EOF
    else
	${KIND} create cluster --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  podSubnet: "192.168.128.0/17"
nodes:
# the control plane node
- role: control-plane
- role: worker
- role: worker
- role: worker
EOF
    fi
    kind_rc=$?
    if ${TEST_RETRY:-false} || test "${kind_rc}" != 0; then
	${KIND} delete cluster
	TEST_RETRY=false	# Only force a retry test once.
	continue
    fi
    break
done
set -e
test "${kind_rc}" = 0

${kubectl} get no -o wide
${kubectl} get po --all-namespaces -o wide

if dual_stack; then
    echo "Set ipv6 address on each node"
    docker exec kind-control-plane ip -6 a a 2001:20::8/64 dev eth0
    docker exec kind-worker ip -6 a a 2001:20::1/64 dev eth0
    docker exec kind-worker2 ip -6 a a 2001:20::2/64 dev eth0
    docker exec kind-worker3 ip -6 a a 2001:20::3/64 dev eth0
    echo
fi

# Add a stable loopback address on kind-control-plane.  See
# https://tigera.atlassian.net/browse/BPF-755 for the detail behind
# this.  In summary, when calico-node (or kubectl ...) connects to an
# API server on the same node, there's a MASQUERADE in the path that
# likes choosing our tunnel address, and the API server connection
# will fail if that tunnel address is then removed.  This problem is
# exposed in node(-private) ST that switches between no-overlay, IP-IP
# and VXLAN, and the impacts include:
#
# - Worst case for Felix (inside calico-node) is that it sits there,
#   non-Ready, for 15 minutes, waiting for the connection to time out.
#   (When it does time out, something automatically retries and
#   immediately succeeds.)
#
# - Another scenario is when kubectl is run as part of some
#   teardown/cleanup, after a test case has passed, and the kubectl
#   fails for no apparent reason.  I think this correlates to cases
#   where the calico-nodes had just been restarted to revert to the ST
#   suite's normal overlay mode.
#
# If we add a stable loopback address, as here, MASQUERADE will choose
# that instead of the tunnel address, and thus we avoid the problem of
# the chosen address disappearing.
docker exec kind-control-plane ip a a 10.100.0.1/32 dev lo

load_image kind-control-plane
load_image kind-worker
load_image kind-worker2
load_image kind-worker3

for image in calico/cni:master calico/pod2daemon-flexvol:master; do
    docker pull ${image}
    rm -f image.tar
    docker save --output image.tar ${image}
    for node in kind-control-plane kind-worker kind-worker2 kind-worker3; do
	docker cp image.tar ${node}:/image.tar
	docker exec -t ${node} ctr -n=k8s.io images import /image.tar
	docker exec -t ${node} rm /image.tar
    done
done

# Install pull secret so we can pull the right calicoctl.
${kubectl} -n kube-system create secret generic cnx-pull-secret \
   --from-file=.dockerconfigjson=${GCR_IO_PULL_SECRET} \
   --type=kubernetes.io/dockerconfigjson

echo "Install Calico and Calicoctl"
cp $TEST_DIR/infra/calico-kdd.yaml $TEST_DIR/infra/calico.yaml
update_calico_manifest $TEST_DIR/infra/calico.yaml
${kubectl} apply -f $TEST_DIR/infra/calico.yaml
# Install Calicoctl on master node, avoid network disruption during bgp configuration.
cat ${TEST_DIR}/infra/calicoctl.yaml | \
    sed 's,beta.kubernetes.io/os: linux,beta.kubernetes.io/os: linux\n  nodeName: kind-control-plane,' | \
    ${kubectl} apply -f -
echo

echo "Wait Calico to be ready..."
while ! time ${kubectl} wait pod -l k8s-app=calico-node --for=condition=Ready -n kube-system --timeout=300s; do
    # This happens when no matching resources exist yet,
    # i.e. immediately after application of the Calico YAML.
    sleep 5
done
time ${kubectl} wait pod -l k8s-app=calico-kube-controllers --for=condition=Ready -n kube-system --timeout=300s
time ${kubectl} wait pod -l k8s-app=kube-dns --for=condition=Ready -n kube-system --timeout=300s
time ${kubectl} wait pod calicoctl --for=condition=Ready -n kube-system --timeout=300s
echo "Calico is running."
echo

# Apply the enterprise license.
# FIXME(karthik): Applying the enterprise license here since the test written don't test for invalid or no license.
# Once such tests are added, this will have to move into the test itself.
${kubectl} exec -i -n kube-system calicoctl -- /calicoctl apply -f - < ${TSEE_TEST_LICENSE}

function test_connection() {
    local svc="webserver-ipv$1"
    output=$(${kubectl} exec client -- wget $svc -T 5 -O -)
    echo $output
    if [[ $output != *test-webserver* ]]; then
	echo "connection to $svc service failed"
	exit 1
    fi
}

echo "Install MetalLB controller for allocating LoadBalancer IPs"
${kubectl} create ns metallb-system
${kubectl} apply -f $TEST_DIR/infra/metallb.yaml
${kubectl} apply -f $TEST_DIR/infra/metallb-config.yaml

# Create and monitor a test webserver service for dual stack.
echo "Create test-webserver deployment..."
${kubectl} apply -f tests/k8st/infra/test-webserver.yaml

if dual_stack; then
    # Create and monitor a test webserver service for dual stack.
    echo "Create test-webserver deployment..."
    ${kubectl} apply -f tests/k8st/infra/test-webserver.yaml

    echo "Wait for client and webserver pods to be ready..."
    while ! time ${kubectl} wait pod -l pod-name=client --for=condition=Ready --timeout=300s; do
	sleep 5
    done
    while ! time ${kubectl} wait pod -l app=webserver --for=condition=Ready --timeout=300s; do
	sleep 5
    done
    echo "client and webserver pods are running."
    echo

    ${kubectl} get po --all-namespaces -o wide
    ${kubectl} get svc

    # Run ipv4 ipv6 connection test
    test_connection 4
    test_connection 6
fi
