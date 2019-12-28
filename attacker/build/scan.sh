#!/bin/bash

while true; do
	# We grab the IP of the running pod
	export IP_LIST=$(ip addr | awk '/inet / {print $2}'| cut -d / -f1 |tail -n 1)/24
	# We nmap the subnet, looking for hosts thats up
	export HOST_LIST=$(nmap -n -sn $IP_LIST | awk '/for /{print $5}')
	# We nmap the ports of the running hosts
	export PORT_LIST=$(nmap -Pn $HOST_LIST)
	echo $PORT_LIST
done
