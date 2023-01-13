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

# IPTABLES_BACKEND may be set to "legacy" (default) or "nft".  The
# "legacy" default should always work, until Linux is eventually
# modified not to support the legacy mode any more.
: ${IPTABLES_BACKEND:=legacy}
iptables=iptables-${IPTABLES_BACKEND}

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
${iptables} -t nat -A POSTROUTING -j MASQUERADE

echo Configure vxlan tunnel device
ip link add vxlan0 type vxlan id $EGRESS_VXLAN_VNI dstport $EGRESS_VXLAN_PORT dev eth0 || printf " (and that's fine)"
ip link set vxlan0 address $MAC
ip link set vxlan0 up

echo Configure network settings
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/vxlan0/rp_filter
