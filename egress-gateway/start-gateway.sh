#!/bin/sh

# Script to start egress gateway pod

set -e

# Capture the usual signals and exit from the script
trap 'echo "INT received, simply exiting..."; exit 0' INT
trap 'echo "TERM received, simply exiting..."; exit 0' TERM
trap 'echo "HUP received, simply exiting..."; exit 0' HUP

: ${DAEMON_SOCKET_PATH:=/var/run/calico/nodeagent/socket}

if [ -z "$EGRESS_POD_IP" ]
then
    echo "EGRESS_POD_IP not defined."
    exit 1
fi

if [ -z "$LOG_SEVERITY" ]
then
    echo "LOG_SEVERITY not defined."
    exit 1
fi

if [ -z "$EGRESS_VXLAN_VNI" ]
then
    echo "EGRESS_VXLAN_VNI not defined."
    exit 1
fi

echo Egress gateway starting...
/egressd start $EGRESS_POD_IP --log-severity $LOG_SEVERITY --vni $EGRESS_VXLAN_VNI --socket-path $DAEMON_SOCKET_PATH
