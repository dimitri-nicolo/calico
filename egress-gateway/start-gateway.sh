#!/bin/sh

# Script to start egress gateway pod

set -e

# Capture the usual signals and exit from the script
trap 'echo "INT received, simply exiting..."; exit 0' INT
trap 'echo "TERM received, simply exiting..."; exit 0' TERM
trap 'echo "HUP received, simply exiting..."; exit 0' HUP

: ${EGRESS_VXLAN_VNI:=4097}

: ${DAEMON_LOG_SEVERITY:=info}

: ${DAEMON_SOCKET_PATH:=/var/run/nodeagent/socket}

if [ -z "$EGRESS_POD_IP" ]
then
    echo "EGRESS_POD_IP not defined."
    exit 1
fi

echo Egress gateway starting...
/egressd start $EGRESS_POD_IP --log-severity $DAEMON_LOG_SEVERITY --vni $EGRESS_VXLAN_VNI --socket-path $DAEMON_SOCKET_PATH
