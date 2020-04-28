#!/bin/sh

# Script to start egress gateway pod

set -e

# Capture the usual signals and exit from the script
trap 'echo "INT received, simply exiting..."; exit 0' INT
trap 'echo "TERM received, simply exiting..."; exit 0' TERM
trap 'echo "HUP received, simply exiting..."; exit 0' HUP

# Check environment variables
if [ -z "$EGRESS_VXLAN_VNI" ]
then
    EGRESS_VXLAN_VNI=4097
fi 

if [ -z "$EGRESS_VXLAN_PORT" ]
then
    EGRESS_VXLAN_PORT=4790
fi 

if [ -z "$EGRESS_POD_IP" ]
then
    echo "EGRESS_POD_IP not defined."
    exit 1
fi

echo Egress VXLAN VNI: $EGRESS_VXLAN_VNI  Egress VXLAN PORT: $EGRESS_VXLAN_PORT

value=`echo $EGRESS_POD_IP | awk -F. '{print $1}'`
ETH_A=`printf '%02x' $value`
value=`echo $EGRESS_POD_IP | awk -F. '{print $2}'`
ETH_B=`printf '%02x' $value`
value=`echo $EGRESS_POD_IP | awk -F. '{print $3}'`
ETH_C=`printf '%02x' $value`
value=`echo $EGRESS_POD_IP | awk -F. '{print $4}'`
ETH_D=`printf '%02x' $value`
MAC=`echo "a2:2a:$ETH_A:$ETH_B:$ETH_C:$ETH_D"`

echo Pod IP: $EGRESS_POD_IP  MAC: $MAC

echo Configure iptable rules
iptables -t nat -A POSTROUTING -j MASQUERADE

echo Configure vxlan tunnel device
ip link add vxlan0 type vxlan id $EGRESS_VXLAN_VNI dstport $EGRESS_VXLAN_PORT dev eth0
ip link set vxlan0 address $MAC
ip link set vxlan0 up

echo Configure network settings
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/vxlan0/rp_filter

echo Egress gateway pod is running...
tail -f /dev/null
