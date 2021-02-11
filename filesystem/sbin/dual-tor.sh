#!/bin/bash

# Do setup for a dual ToR node, then run as the "early BGP" daemon
# until calico-node's BIRD can take over.

set -ex

ROUTER_ID=`ip -4 -o address | while read num intf inet addr; do
    case $intf in
	# Allow for "eth0", "ens5", "enp0s3" etc.; avoid "lo" and
	# "docker0".
	e*)
	    echo ${addr%/*}
	    break
	    ;;
    esac
done`

if [ -z "$ROUTER_ID" ]; then
    ROUTER_ID='127.0.0.1'
fi

# Generate bird.conf and bird6.conf.
dual_tor=/etc/calico/dual-tor
bird_cfg=/etc/calico/confd/config
sed "s/BIRD_ROUTERID/${ROUTER_ID}/g" ${dual_tor}/bird.conf > ${bird_cfg}/bird.cfg
sed "s/BIRD_ROUTERID/${ROUTER_ID}/g" ${dual_tor}/bird6.conf > ${bird_cfg}/bird6.cfg

# Ensure that /var/run/calico (which is where the control socket file
# will be) exists.
mkdir -p /var/run/calico

# Shell implementaton of `ipcalc -n`...
ipcalc_n()
{
    prefix_len=${1#*/}
    addr=${1%/*}
    b1=${addr%%.*}
    r1=${addr#*.}
    b2=${r1%%.*}
    r2=${r1#*.}
    b3=${r2%%.*}
    b4=${r2#*.}
    if [ $prefix_len -lt 8 ]; then
	shft=$(( 8 - prefix_len ))
	b1=$(( (b1 >> shft) << shft ))
	b2=0
	b3=0
	b4=0
    elif [ $prefix_len -lt 16 ]; then
	shft=$(( 16 - prefix_len ))
	b2=$(( (b2 >> shft) << shft ))
	b3=0
	b4=0
    elif [ $prefix_len -lt 24 ]; then
	shft=$(( 24 - prefix_len ))
	b3=$(( (b3 >> shft) << shft ))
	b4=0
    elif [ $prefix_len -lt 32 ]; then
	shft=$(( 32 - prefix_len ))
	b4=$(( (b4 >> shft) << shft ))
    fi
    echo ${b1}.${b2}.${b3}.${b4}
}

# Given an address and interface in the same subnet as a ToR address/prefix,
# update the address in the ways that we need for dual ToR operation, and ensure
# that we still have the routes that we'd expect through that interface.
try_update_nic_addr()
{
    addr=$1
    intf=$2
    tor_addr=$3
    change_to_scope_link=$4

    # Calculate the prefix length and IP network of the given address.
    subnet_prefix_len=${addr#*/}
    subnet_network=`ipcalc_n ${addr}`

    # Calculate the IP network of the subnet that $tor_addr is in (assuming the
    # same prefix length as for the given NIC address).
    tor_network=`ipcalc_n ${tor_addr}/${subnet_prefix_len}`

    # If the networks are the same...
    if [ "$subnet_network" = "$tor_network" ]; then

	if $change_to_scope_link; then
	    # Delete the given address and re-add it with scope link.
	    ip address del $addr dev $intf
	    ip address add $addr dev $intf scope link
	fi

	# Ensure that the subnet route is present.  (This 'ip route add' will
	# fail if the same route is already present.)
	ip route add ${subnet_network}/${subnet_prefix_len} dev $intf || true

	# Try to add a default route via the ToR.  (This 'ip route add' will
	# fail if we already have a default route, e.g. via the other ToR.)
	ip route add default via ${tor_addr} || true

    fi
}

# There must be a file mapped in at /etc/calico-dual-tor/details.sh
# that defines a `get_dual_tor_details` function.  Given one of the
# NIC-specific addresses for this node, this function must output
# variable settings for this node's stable address and AS number, and
# for the ToR addresses to peer with, for example like this:
#
# DUAL_TOR_STABLE_ADDRESS=172.31.20.3
# DUAL_TOR_AS_NUMBER=65002
# DUAL_TOR_PEERING_ADDRESS_1=172.31.21.100
# DUAL_TOR_PEERING_ADDRESS_2=172.31.22.100
#
# Note, the shell include code should not have any effects other than
# defining the required `get_dual_tor_details` function.
. /etc/calico-dual-tor/details.sh

echo "NIC-specific address is $ROUTER_ID"
eval `get_dual_tor_details $ROUTER_ID`
echo "Stable (loopback) address is $DUAL_TOR_STABLE_ADDRESS"
echo "AS number is $DUAL_TOR_AS_NUMBER"
echo "ToR addresses are $DUAL_TOR_PEERING_ADDRESS_1 and $DUAL_TOR_PEERING_ADDRESS_2"

# Configure the stable address.
ip address add ${DUAL_TOR_STABLE_ADDRESS}/32 dev lo

# Generate BIRD peering config.
mkdir -p ${bird_cfg}/bird
cat >${bird_cfg}/bird/peers.conf <<EOF
filter stable_address_only {
  if ( net = ${DUAL_TOR_STABLE_ADDRESS}/32 ) then { accept; }
  reject;
}
template bgp tors {
  description "Connection to ToR";
  local as $DUAL_TOR_AS_NUMBER;
  direct;
  gateway recursive;
  import all;
  export filter stable_address_only;
  add paths on;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
  next hop self;
}
protocol bgp tor1 from tors {
  neighbor $DUAL_TOR_PEERING_ADDRESS_1 as $DUAL_TOR_AS_NUMBER;
}
protocol bgp tor2 from tors {
  neighbor $DUAL_TOR_PEERING_ADDRESS_2 as $DUAL_TOR_AS_NUMBER;
}
EOF

# Change interface-specific addresses to be scope link.
ip -4 -o address | while read num intf inet addr rest; do
    case $intf in
	# Allow for "eth0", "ens5", "enp0s3" etc.; avoid "lo" and
	# "docker0".
	e* )
	    try_update_nic_addr $addr $intf $DUAL_TOR_PEERING_ADDRESS_1 true
	    try_update_nic_addr $addr $intf $DUAL_TOR_PEERING_ADDRESS_2 true
	    ;;
    esac
done

# Use multiple ECMP paths based on hashing 5-tuple.
echo 1 > /proc/sys/net/ipv4/fib_multipath_hash_policy
echo 1 > /proc/sys/net/ipv4/fib_multipath_use_neigh

# Loop deciding whether to run early BIRD or not.
sv down bird
early_bird_running=false
while true; do
    # /proc/net/tcp shows TCP listens and connections, and 00000000:00B3, if
    # present, indicates a process listening on port 179.  (179 = 0xB3)
    if grep 00000000:00B3 /proc/net/tcp; then
	# Calico BIRD is running.
	if $early_bird_running; then
	    sv down bird
	    early_bird_running=false
	fi
    else
	# Calico BIRD is not running.
	if ! $early_bird_running; then
	    # Start early bird service
	    sv up bird
	    early_bird_running=true
	fi
    fi
    # Ensure subnet routes are present.
    ip -4 -o address | while read num intf inet addr rest; do
	case $intf in
	    # Allow for "eth0", "ens5", "enp0s3" etc.; avoid "lo" and
	    # "docker0".
	    e* )
		try_update_nic_addr $addr $intf $DUAL_TOR_PEERING_ADDRESS_1 false
		try_update_nic_addr $addr $intf $DUAL_TOR_PEERING_ADDRESS_2 false
		;;
	esac
    done
    sleep 10
done
