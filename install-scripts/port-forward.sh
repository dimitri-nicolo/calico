#!/bin/sh

echo "Portforward 9443 and 5601 to access TSEE UI and Kibana"
kubectl port-forward -n tigera-manager svc/tigera-manager 9443 &
kubectl port-forward -n tigera-kibana svc/tigera-secure-kb-http 5601 &

