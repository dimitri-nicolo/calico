#!/bin/bash -e

# test directory.
TEST_DIR=./tests/k8st

# kubectl binary.
: ${kubectl:=../hack/test/kind/kubectl}

# Normally, cleanup any leftover state, then setup, then test.
: ${STEPS:=cleanup setup}

nodeIP=$(${kubectl} get node kind-worker -o jsonpath='{.status.addresses[0].address}')
echo kind-worker node ip $nodeIP

function add_calico_resources() {
  # Setup BGPPeers for each router.
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
  name: peer-a1-v6
spec:
  nodeSelector: "egress == 'true'"
  peerIP: fd00:0:1234:1111::1
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
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-b1-v6
spec:
  nodeSelector: "egress == 'true'"
  peerIP: fd00:0:1234:2222::1
  asNumber: 64512
  sourceAddress: None
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-b2
spec:
  nodeSelector: "egress == 'true'"
  peerIP: 172.31.21.3
  asNumber: 64512
  sourceAddress: None
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-b2-v6
spec:
  nodeSelector: "egress == 'true'"
  peerIP: fd00:0:1234:2222::3
  asNumber: 64512
  sourceAddress: None
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-c1
spec:
  nodeSelector: "egress == 'true'"
  peerIP: 172.31.31.1
  asNumber: 64512
  sourceAddress: None
---
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: peer-c1-v6
spec:
  nodeSelector: "egress == 'true'"
  peerIP: fd00:0:1234:3333::1
  asNumber: 64512
  sourceAddress: None
EOF

    # Label and annotate nodes.
    ${kubectl} label node kind-worker egress=true --overwrite

    # This is necessary for the correct node ip to be used on the node-node mesh.
    ${kubectl} set env daemonset/calico-node -n kube-system IP_AUTODETECTION_METHOD=kubernetes-internal-ip
    ${kubectl} set env daemonset/calico-node -n kube-system IP6_AUTODETECTION_METHOD=kubernetes-internal-ip

    # Restart the calico-node daemonset to apply the changes.
    ${kubectl} rollout restart daemonset/calico-node -n kube-system
}

function do_setup {
    # Fix rp_filter setting.
    sudo sysctl -w net.ipv4.conf.all.rp_filter=1

    # Create docker networks for this topology:
    #
    #    +---------+            +---------+                           +---------+
    #    | nginx-a |            | nginx-b |                           | nginx-c |
    #    +---------+            +---------+                           +---------+
    #         | .1                | .1                                    |.1
    #         |                   |                                       |
    #         | 172.31.41         | 172.31.41                             | 172.31.41
    #         | 'servernetA'      | 'servernetB'                          | 'servernetC'
    #         |                   |                                       |
    #         |                   |                                       |
    #         |.2                 |.2                                     |.2
    #    +---------+         +---------+         +---------+          +---------+
    #    | bird-a1 |         | bird-b1 |---------| bird-b2 |          | bird-c1 |
    #    +---------+         +---------+         +---------+          +---------+
    #         |.1                 |.1                 |.3                 |.1
    #         |                   |                   |                   |
    #         |                   |--------------------                   |
    #         | 172.31.11         | 172.31.21                             | 172.31.31
    #         |  'enetA'          |  'enetB'                              |  'enetC'
    #         |                   |                                       |
    #         |.4                 |.4                                     |.4
    #  +---------------------------------------------------------------------------------+
    #  |                                                                                 |
    #  +---------------------------------------------------------------------------------+
    #                  kind-worker (node ip 172.24.0.x)

    # Create external networks
    docker network create --subnet=172.31.11.0/24 --gateway 172.31.11.2 --ipv6 --subnet=fd00:0:1234:1111::/64 --gateway fd00:0:1234:1111::2 enetA
    docker network create --subnet=172.31.21.0/24 --gateway 172.31.21.2 --ipv6 --subnet=fd00:0:1234:2222::/64 --gateway fd00:0:1234:2222::2 enetB
    docker network create --subnet=172.31.31.0/24 --gateway 172.31.31.2 --ipv6 --subnet=fd00:0:1234:3333::/64 --gateway fd00:0:1234:3333::2 enetC

    # Create routers on external networks
    docker run -d --privileged --net=enetA --ip=172.31.11.1 --ip6=fd00:0:1234:1111::1 --name=bird-a1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.1 --ip6=fd00:0:1234:2222::1 --name=bird-b1 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetB --ip=172.31.21.3 --ip6=fd00:0:1234:2222::3 --name=bird-b2 ${ROUTER_IMAGE}
    docker run -d --privileged --net=enetC --ip=172.31.31.1 --ip6=fd00:0:1234:3333::1 --name=bird-c1 ${ROUTER_IMAGE}

    # Connect kind-worker to networks
    docker network connect --ip=172.31.11.4 --ip6=fd00:0:1234:1111::4 enetA kind-worker
    docker network connect --ip=172.31.21.4 --ip6=fd00:0:1234:2222::4 enetB kind-worker
    docker network connect --ip=172.31.31.4 --ip6=fd00:0:1234:3333::4 enetC kind-worker

    # Configure Router end of cluster node peerings.
    # Note default route will be filtered out for each router. 
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/nodes-enetA.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = 0.0.0.0/0 then reject;
      accept;
  };
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

    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird6/nodes-enetA-v6.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = ::/0 then reject;
      accept;
  };
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
  neighbor fd00:0:1234:1111::4 as 64512;
}
EOF
    docker exec bird-a1 birdcl6 configure

    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird/nodes-enetB.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = 0.0.0.0/0 then reject;
      accept;
  };
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

    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /etc/bird6/nodes-enetB-v6.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = ::/0 then reject;
      accept;
  };
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
  neighbor fd00:0:1234:2222::4 as 64512;
}
EOF
    docker exec bird-b1 birdcl6 configure

     cat <<EOF | docker exec -i bird-b2 sh -c "cat > /etc/bird/nodes-enetB.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = 0.0.0.0/0 then reject;
      accept;
  };
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
    docker exec bird-b2 birdcl configure

    cat <<EOF | docker exec -i bird-b2 sh -c "cat > /etc/bird6/nodes-enetB-v6.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = ::/0 then reject;
      accept;
  };
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
  neighbor fd00:0:1234:2222::4 as 64512;
}
EOF
    docker exec bird-b2 birdcl6 configure

         cat <<EOF | docker exec -i bird-c1 sh -c "cat > /etc/bird/nodes-enetC.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = 0.0.0.0/0 then reject;
      accept;
  };
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
  neighbor 172.31.31.4 as 64512;
}
EOF
    docker exec bird-c1 birdcl configure

    cat <<EOF | docker exec -i bird-c1 sh -c "cat > /etc/bird6/nodes-enetC-v6.conf"
template bgp nodes {
  description "Connection to BGP peer";
  local as 64512;
  direct;
  gateway recursive;
  import all;
  export filter {
      if net = ::/0 then reject;
      accept;
  };
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
  neighbor fd00:0:1234:3333::4 as 64512;
}
EOF
    docker exec bird-c1 birdcl6 configure

    # Create BGPConfiguration, BGPPeers etc.
    add_calico_resources

    # Add some routes for each routers
    docker exec bird-a1 ip route add blackhole 10.233.11.8
    docker exec bird-b1 ip route add blackhole 10.233.21.8
    docker exec bird-b2 ip route add blackhole 10.233.21.9
    docker exec bird-c1 ip route add blackhole 10.233.31.8

    # Add nginx docker containers to each external network (as docker-in-docker on the external bird containers)
    docker exec bird-a1 docker network create --subnet 172.31.41.0/24 --gateway 172.31.41.2 servernetA --ipv6 --subnet=fd00:0:1234:4444::/64 --gateway fd00:0:1234:4444::2
    docker exec bird-a1 mkdir -p /tmp/nginx
    cat <<EOF | docker exec -i bird-a1 sh -c "cat > /tmp/nginx/index.html"
server A
EOF
    docker exec bird-a1 chmod -R 0755 /tmp/nginx/
    docker exec bird-a1 docker run --network servernetA --ip 172.31.41.1 -d --name nginx-a -v /tmp/nginx/:/usr/share/nginx/html:ro nginx

    docker exec bird-b1 docker network create --subnet 172.31.41.0/24 --gateway 172.31.41.2 servernetB --ipv6 --subnet=fd00:0:1234:4444::/64 --gateway fd00:0:1234:4444::2
    docker exec bird-b1 mkdir -p /tmp/nginx
    cat <<EOF | docker exec -i bird-b1 sh -c "cat > /tmp/nginx/index.html"
server B
EOF
    docker exec bird-b1 chmod -R 0755 /tmp/nginx/
    docker exec bird-b1 docker run --network servernetB --ip 172.31.41.1 -d --name nginx-b -v /tmp/nginx/:/usr/share/nginx/html:ro nginx

    docker exec bird-c1 docker network create --subnet 172.31.41.0/24 --gateway 172.31.41.2 servernetC --ipv6 --subnet=fd00:0:1234:4444::/64 --gateway fd00:0:1234:4444::2
    docker exec bird-c1 mkdir -p /tmp/nginx
    cat <<EOF | docker exec -i bird-c1 sh -c "cat > /tmp/nginx/index.html"
server C
EOF
    docker exec bird-c1 chmod -R 0755 /tmp/nginx/
    docker exec bird-c1 docker run --network servernetC --ip 172.31.41.1 -d --name nginx-c -v /tmp/nginx/:/usr/share/nginx/html:ro nginx
}

function do_cleanup {
    docker rm -f bird-a1 bird-b1 bird-b2 bird-c1

    docker network disconnect enetA kind-worker || true
    docker network disconnect enetB kind-worker || true
    docker network disconnect enetC kind-worker || true
    docker network rm enetA enetB enetC || true

    docker network ls
    docker ps -a
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done
