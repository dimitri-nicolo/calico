#!/bin/bash

# This script updates the manifsts in this directory using helm.
# Values files for the manifests in this directory can be found in 
# ../calico/charts/values.

# Helm binary to use. Default to the one installed by the Makefile.
HELM=${HELM:-../bin/helm}

# Get versions to install.
defaultCalicoVersion=master # TODO
CALICO_VERSION=${CALICO_VERSION:-$defaultCalicoVersion}

defaultOperatorVersion=$(cat ../charts/tigera-operator/values.yaml | grep version: | cut -d" " -f4)
OPERATOR_VERSION=${OPERATOR_VERSION:-$defaultOperatorVersion}

NON_HELM_MANIFEST_IMAGES="calico/apiserver calico/windows calico/ctl calico/csi calico/node-driver-registrar"

echo "Generating manifests for Calico=$CALICO_VERSION and tigera-operator=$OPERATOR_VERSION"

##########################################################################
# Build the operator manifest. 
##########################################################################
cat <<EOF > tigera-operator.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tigera-operator
  labels:
    name: tigera-operator
EOF

${HELM} -n tigera-operator template \
	--include-crds \
	--set installation.enabled=false \
	--set apiServer.enabled=false \
	--set apiServer.enabled=false \
	--set intrusionDetection.enabled=false \
	--set logCollector.enabled=false \
	--set logStorage.enabled=false \
	--set manager.enabled=false \
	--set monitor.enabled=false \
	--set compliance.enabled=false \
	--set tigeraOperator.version=$OPERATOR_VERSION \
	--set calicoctl.tag=$CALICO_VERSION \
	../charts/tigera-operator >> tigera-operator.yaml

##########################################################################
# Build other Tigera operator manifests.
#
# To add a new manifest to this directory, define
# a new values file in ../charts/values/
##########################################################################
VALUES_FILES=$(cd ../charts/values && find . -type f -name "*.yaml")

for FILE in $VALUES_FILES; do
	echo "Generating manifest from charts/values/$FILE"
	# Default to using tigera-operator. However, some manifests use other namespaces instead,
	# as indicated by a comment at the top of the values file of the following form:
	# NS: <namespace-to-use>
	ns=$(cat ../charts/values/$FILE | grep -Po '# NS: \K(.*)')
	${HELM} -n ${ns:-"tigera-operator"} template \
		../charts/tigera-operator \
	        --set version=$CALICO_VERSION \
		-f ../charts/values/$FILE > $FILE
done

##########################################################################
# Build CRDs files used in docs.
##########################################################################
echo "# Tigera Operator and Calico Enterprise CRDs" > crds.yaml
for FILE in $(ls ../charts/tigera-operator/crds); do
        ${HELM} template ../charts/tigera-operator \
                --include-crds \
                --show-only $FILE >> crds.yaml
done
for FILE in $(ls ../charts/tigera-operator/crds/calico); do
        ${HELM} template ../charts/tigera-operator \
                --include-crds \
                --show-only calico/$FILE >> crds.yaml
done

echo "# ECK operator CRDs." > eck-operator-crds.yaml
for FILE in $(ls ../charts/tigera-operator/crds/eck); do
	${HELM} template ../charts/tigera-operator \
                --include-crds \
                --show-only eck/$FILE >> eck-operator-crds.yaml
done

echo "# Prometheus operator CRDs." > prometheus-operator-crds.yaml
for FILE in $(ls ../charts/tigera-prometheus-operator/crds); do
	${HELM} template ../charts/tigera-prometheus-operator \
                --include-crds \
                --show-only $FILE \
		-f ../charts/tigera-operator/values.yaml >> prometheus-operator-crds.yaml
done


##########################################################################
# Build tigera-operator manifests for OCP.
#
# OCP requires resources in their own yaml files, so output to a dir.
# Then do a bit of cleanup to reduce the directory depth to 1.
##########################################################################
${HELM} template --include-crds \
	-n tigera-operator \
	../charts/tigera-operator/ \
	--output-dir ocp \
	--set installation.kubernetesProvider=openshift \
	--set installation.enabled=false \
	--set apiServer.enabled=false \
	--set apiServer.enabled=false \
	--set intrusionDetection.enabled=false \
	--set logCollector.enabled=false \
	--set logStorage.enabled=false \
	--set manager.enabled=false \
	--set monitor.enabled=false \
	--set compliance.enabled=false \
	--set tigeraOperator.version=$OPERATOR_VERSION \
	--set calicoctl.tag=$CALICO_VERSION
# The first two lines are a newline and a yaml separator - remove them.
find ocp/tigera-operator -name "*.yaml" | xargs sed -i -e 1,2d
mv $(find ocp/tigera-operator -name "*.yaml") ocp/ && rm -r ocp/tigera-operator

##########################################################################
# Replace image versions for "static" Calico manifests.
##########################################################################
if [[ $CALICO_VERSION != master ]]; then
echo "Replacing image versions for static manifests"
	for img in $NON_HELM_MANIFEST_IMAGES; do
		echo $img
		find . -type f -exec sed -i "s|$img:[A-Xa-z0-9_.-]*|$img:$CALICO_VERSION|g" {} \;
	done
fi
