#!/bin/bash
#
# This script switches the current operator installation to use the operator
# running in the NEW_NAME namespace.
#
# Before this script is run, the new operator and its associated resources
# should have been deployed to the NEW_NAME namespace.
set -e

NEW_NAME=$1

if [ -z "${NEW_NAME}" ]; then
  echo "Provide a NEW_NAME for the new namespace to switch the current operator to."
  exit 1
fi

copy_resource_to_ns (){
  TYPE=$1
  NAME=$2
  NS=$3

  DATA=$(kubectl get $TYPE -n tigera-operator ${NAME} -o jsonpath="{.data}")
  OWNER_REF=$(kubectl get $TYPE -n tigera-operator ${NAME} -o jsonpath="{.metadata.ownerReferences}")
  kubectl apply -f - <<EOF
{"apiVersion":"v1","kind":"$TYPE",
"data":${DATA},
"metadata":{"name":"${NAME}","namespace":"${NS}",
"ownerReferences": ${OWNER_REF} }}
EOF
}

# Copy over the secrets in the tigera-operator namespace to the new namespace to ensure
# switching the active operator is non-disruptive.
for x in node-certs typha-certs; do
  copy_resource_to_ns Secret $x ${NEW_NAME}
done
copy_resource_to_ns ConfigMap typha-ca ${NEW_NAME}

# Switch the active operator
PATCH_FILE=$(mktemp)
cat <<EOF > ${PATCH_FILE}
{"data":{"active-namespace": "${NEW_NAME}"}}
EOF
kubectl patch configmap -n calico-system active-operator --type=merge --patch-file "${PATCH_FILE}"

# Restart the active operator
kubectl rollout restart deployment -n ${NEW_NAME} tigera-operator
