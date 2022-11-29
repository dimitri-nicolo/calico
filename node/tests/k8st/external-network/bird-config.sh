#!/bin/bash -e

# This script sets up bird.cfg for three external networks and check routes.

: ${kubectl:=../hack/test/kind/kubectl}

: ${STEPS:=cleanup setup}

node="kind-worker"
pod=$(${kubectl} get pod -A -o wide | grep -w ${node} | grep calico-node | awk '{print $2}')

CalicoNodeExec="${kubectl} exec -i $pod -n kube-system -c calico-node -- sh -c"

function do_setup {
  echo Save bird.cfg to bird.cfg.original
  ${CalicoNodeExec} "cp /etc/calico/confd/config/bird.cfg /etc/calico/confd/config/bird.cfg.original"

  echo Write bird config for networks
  cat <<EOF | ${CalicoNodeExec} "cat > /etc/calico/confd/config/networks.cfg"
# ------------- Node-specific peers -------------
# enetA peers -------------
table T_enet_a;
protocol direct D_enet_a {
  debug { states };
  table T_enet_a;
  interface -"cali*", -"kube-ipvs*", "*";
}

protocol kernel K_enet_a {
  learn;
  persist;
  scan time 2;
  device routes yes;
  table T_enet_a;
  kernel table 11;
  import all;
  export filter {
      print "route: ", net, ", ", from, ", ", proto, ", ", bgp_next_hop;
      if proto = "Node_172_31_11_1" then accept;
      reject;
  };
}

# For peer /host/kind-worker/peer_v4/172.31.11.1
protocol bgp Node_172_31_11_1 from bgp_template {
  neighbor 172.31.11.1 as 64512;
  ttl security off;
  direct;
  gateway recursive;
  table T_enet_a;
}

# enetA peers -------------

# enetB peers -------------
table T_enet_b;
protocol direct D_enet_b {
  debug { states };
  table T_enet_b;
  interface -"cali*", -"kube-ipvs*", "*";
}

protocol kernel K_enet_b {
  learn;
  persist;
  scan time 2;
  device routes yes;
  table T_enet_b;
  kernel table 21;
  import all;
  export filter {
      print "route: ", net, ", ", from, ", ", proto, ", ", bgp_next_hop;
      if proto = "Node_172_31_21_1" then accept;
      if proto = "Node_172_31_21_3" then accept;
      reject;
  };
}

# For peer /host/kind-worker/peer_v4/172.31.21.1
protocol bgp Node_172_31_21_1 from bgp_template {
  neighbor 172.31.21.1 as 64512;
  ttl security off;
  direct;
  gateway recursive;
  table T_enet_b;
}

# For peer /host/kind-worker/peer_v4/172.31.21.3
protocol bgp Node_172_31_21_3 from bgp_template {
  neighbor 172.31.21.3 as 64512;
  ttl security off;
  direct;
  gateway recursive;
  table T_enet_b;
}

# enetB peers -------------

# enetC peers -------------
table T_enet_c;
protocol direct D_enet_c {
  debug { states };
  table T_enet_c;
  interface -"cali*", -"kube-ipvs*", "*";
}

protocol kernel K_enet_d {
  learn;
  persist;
  scan time 2;
  device routes yes;
  table T_enet_c;
  kernel table 31;
  import all;
  export filter {
      print "route: ", net, ", ", from, ", ", proto, ", ", bgp_next_hop;
      if proto = "Node_172_31_31_1" then accept;
      reject;
  };
}

# For peer /host/kind-worker/peer_v4/172.31.31.1
protocol bgp Node_172_31_31_1 from bgp_template {
  neighbor 172.31.31.1 as 64512;
  ttl security off;
  direct;
  gateway recursive;
  table T_enet_c;
}

# enetC peers -------------
EOF

  echo Remove all node-specific peers
  ${CalicoNodeExec} "sed -i '/^# ------------- Node-specific peers -------------/,$ d' /etc/calico/confd/config/bird.cfg"

  echo Add networks.cfg
  cat <<EOF | ${CalicoNodeExec} "cat >> /etc/calico/confd/config/bird.cfg"
# ------------- Node-specific peers -------------
include "networks.cfg";
EOF

  echo Reload bird config
  ${CalicoNodeExec} "sv hup bird"

  echo Sleep 10 seconds...
  sleep 10

  echo; echo bird show protocols:
  ${CalicoNodeExec} "birdcl -s /var/run/calico/bird.ctl show protocols"

  for t in 11 21 31; do
    echo; echo show routes on table $t:
    ${CalicoNodeExec} "ip route show table $t"
  done
}

function do_cleanup {
  echo Restore bird.cfg
  ${CalicoNodeExec} "cp /etc/calico/confd/config/bird.cfg.original /etc/calico/confd/config/bird.cfg" || true

  echo Remove networks.cfg
  ${CalicoNodeExec} "rm /etc/calico/confd/config/networks.cfg" || true
}

# Execute requested steps.
for step in ${STEPS}; do
    eval do_${step}
done
