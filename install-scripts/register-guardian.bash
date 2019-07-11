
if [ $# != 2 ]; then
  cat >&2 <<EOF
Usage: register-guardian.bash <id> <displayName>
EOF
  exit 1
fi

id="$1"
displayName="$2"

IP=$(kubectl get svc cnx-voltron-server -n calico-monitoring -o=jsonpath='{.spec.clusterIP}')
PORT=$(kubectl get svc cnx-voltron-server -n calico-monitoring -o=jsonpath='{.spec.ports[?(@.name=="mgmt")].port}')
curl --insecure "https://${IP}:${PORT}/voltron/api/clusters" -X PUT -H "Content-type: application/json" -d '{"id":"'"${id}"'", "displayName":"'"${displayName}"'"}' -o "guardian-${id}.yaml"

