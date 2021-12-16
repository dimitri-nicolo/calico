#!/bin/bash
#
# This script modifies the tigera-operator.yaml manifest so that
# it can be used to deploy new resources for an operator-based installation
# upgrade.
#
# In most cases, this script simply updates the .metadata.name and
# .metadata.namespace fields from "tigera-operator" to the new name.
#
# This script requires docker in order to run.
set -e
NEW_NS=$1

if [ -z "${NEW_NS}" ]; then
  echo "Provide a NEW_NS for the new namespace to switch the current operator to."
  echo "The name must conform to Kubernetes label naming standards."
  exit 1
fi

MANIFEST=tigera-operator.yaml
NEW_MANIFEST=tigera-operator-new.yaml

if [ ! -f ${MANIFEST} ]; then
  echo "${MANIFEST} not found, download it with:"
  echo ""
  echo "  curl https://docs.tigera.io/manifests/tigera-operator.yaml -o ${MANIFEST}"
  echo ""
  echo "Then re-run this script to modify ${MANIFEST}."
  exit 1
fi

yq() {
docker run \
  --user $(id -u):$(id -g) \
  --volume `pwd`:/workdir:rw \
  --env NEW_NS=${NEW_NS} \
  --rm mikefarah/yq:4.16.1 "$@"
}


if [ -f ${NEW_MANIFEST} ]; then
  echo "The file ${NEW_MANIFEST} already exists; remove it and retry."
  exit 1
fi

cp ${MANIFEST} ${NEW_MANIFEST}
echo "Copied ${MANIFEST} to ${NEW_MANIFEST}"

echo "Updating ${NEW_MANIFEST} to use ${NEW_NS}..."
# Update manifest to use whatever formatting yq uses.
yq eval --inplace '.' /workdir/tigera-operator.yaml

# Update operator namespace
yq eval --inplace '
  select(.kind == "Namespace" and .metadata.name == "tigera-operator").metadata.name |= env(NEW_NS) |
  select(.kind == "Namespace" and .metadata.name == env(NEW_NS)).metadata.labels.name |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Update operator deployment's namespace
yq eval --inplace '
  select(.kind == "Deployment" and .metadata.name == "tigera-operator").metadata.namespace |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Update service account's namespace
yq eval --inplace '
  select(.kind == "ServiceAccount" and .metadata.name == "tigera-operator").metadata.namespace |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Update podsecuritypolicy's name
yq eval --inplace '
  select(.kind == "PodSecurityPolicy" and .metadata.name == "tigera-operator").metadata.name |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Update clusterrole's name and the podsecuritypolicies use rule
# Note: for the second yq command, the LHS is surrounded with brackets since the pipe has lower precence.
yq eval --inplace '
  select(.kind == "ClusterRole" and .metadata.name == "tigera-operator").metadata.name |= env(NEW_NS) |
  ( select(.kind == "ClusterRole" and .metadata.name == env(NEW_NS)).rules[] |
    select(.apiGroups[0] == "policy" and .resources[0] == "podsecuritypolicies" and .resourceNames[0] == "tigera-operator")
  ).resourceNames[0] |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Update clusterrolebinding's namespace and subject namespace
yq eval --inplace '
  select(.kind == "ClusterRoleBinding" and .metadata.name == "tigera-operator").metadata.name |= env(NEW_NS) |
  select(.kind == "ClusterRoleBinding" and .metadata.name == env(NEW_NS)).subjects[0].namespace |= env(NEW_NS) |
  select(.kind == "ClusterRoleBinding" and .metadata.name == env(NEW_NS)).roleRef.name |= env(NEW_NS)
  '  /workdir/${NEW_MANIFEST}

# Remove '{}' cruft leftover by yq
sed -i 's|^{}$||g' ${NEW_MANIFEST}

echo "Finished updating ${NEW_MANIFEST} to use ${NEW_NS}"

