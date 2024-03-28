#!/bin/sh

# This script initializes settings necessary for EGW pods to run.
# The reason for having a separate init script is that, this script
# runs as part of an init container, which runs in privileged mode.
# This allows us to run the EGW pods as non-privileged.

set -e

# Capture the usual signals and exit from the script
trap 'echo "INT received, simply exiting..."; exit 0' INT
trap 'echo "TERM received, simply exiting..."; exit 0' TERM
trap 'echo "HUP received, simply exiting..."; exit 0' HUP

if [ -z "$EGRESS_POD_IP" ]
then
    echo "EGRESS_POD_IP not defined."
    exit 1
fi

if [ -z "$EGRESS_VXLAN_VNI" ]
then
    echo "EGRESS_VXLAN_VNI not defined."
    exit 1
fi

if [ -z "$EGRESS_VXLAN_PORT" ]
then
    echo "EGRESS_VXLAN_PORT not defined."
    exit 1
fi

MAC=`echo $EGRESS_POD_IP | awk -F. '{printf "a2:2a:%02x:%02x:%02x:%02x", $1, $2, $3, $4}'`

echo Egress VXLAN VNI: $EGRESS_VXLAN_VNI  VXLAN PORT: $EGRESS_VXLAN_PORT VXLAN MAC: $MAC Pod IP: $EGRESS_POD_IP

echo Configure iptable rules

# IPTABLES_BACKEND may be set to "legacy" or "nft".  If not defined
# try to auto-detect nft or legacy.

if [ "$IPTABLES_BACKEND" ]
then
    echo "IPTABLES_BACKEND set to $IPTABLES_BACKEND"
    iptables-${IPTABLES_BACKEND} -t nat -A POSTROUTING -j MASQUERADE
    IPTABLES_BINARY="iptables-${IPTABLES_BACKEND}"
elif iptables-nft -t nat -A POSTROUTING -j MASQUERADE
then
    IPTABLES_BINARY="iptables-nft"
    echo "Successfully configured iptables with iptables-nft."
elif iptables-legacy -t nat -A POSTROUTING -j MASQUERADE
then
    IPTABLES_BINARY="iptables-legacy"
    echo "Successfully configured iptables with iptables-legacy."
else
    echo "Failed to configure iptables (tried both nft and legacy)."
    exit 1
fi

echo Configure vxlan tunnel device
ip link add vxlan0 type vxlan id $EGRESS_VXLAN_VNI dstport $EGRESS_VXLAN_PORT dev eth0 || printf " (and that's fine)"
ip link set vxlan0 address $MAC
ip link set vxlan0 up

echo "Adding iptables MSS clamping rules on interface eth0"
# Detect vxlan0 MTU
VXLAN_MTU=`awk '{print $1}' /sys/class/net/vxlan0/mtu`

# Calculate MSS_CLAMP_VALUE=VXLAN_MTU - 40 (IPv4 header len + TCP header len)
MSS_CLAMP_VALUE="$(($VXLAN_MTU - 40))"
echo "Detected vxlan0 MTU=$VXLAN_MTU. Clamping MSS value to $MSS_CLAMP_VALUE"
args="FORWARD -t mangle -o eth0 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss $MSS_CLAMP_VALUE"
$IPTABLES_BINARY -C $args 2>/dev/null || $IPTABLES_BINARY -A $args

echo Configure network settings
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/vxlan0/rp_filter
