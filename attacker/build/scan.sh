#!/bin/bash

while true; do
	# We grab the IP of the running pod
	export IP_LIST=$(ip addr | awk '/inet / {print $2}'| cut -d / -f1 |tail -n 1)/24
	# We nmap the subnet, looking for hosts thats up
	export HOST_LIST=$(nmap -n -sn $IP_LIST | awk '/for /{print $5}')
	# We nmap the ports of the running hosts
	export PORT_LIST=$(nmap -Pn $HOST_LIST)
	echo $PORT_LIST
	curl https://tigera-dashboard-internal-service.tigera-internal.svc.cluster.local -kL
	curl http://tigera-dashboard-internal-debug.tigera-internal.svc.cluster.local:8888
	curl http://tigera-internal-backend.tigera-internal.svc.cluster.local:3306
	nmap -Pn --script=mysql-enum tigera-internal-backend.tigera-internal.svc.cluster.local
done
