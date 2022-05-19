#!/bin/bash -e

docker exec "${APISERVER_NAME}" kubectl apply -f "/go/src/${PACKAGE_NAME}/test/role.yaml"
docker exec "${APISERVER_NAME}" kubectl apply -f "/go/src/${PACKAGE_NAME}/test/token-binding.yaml"

docker exec "${APISERVER_NAME}" kubectl apply -f "/go/src/${PACKAGE_NAME}/test/pip/roles.yaml"
docker exec "${APISERVER_NAME}" kubectl apply -f "/go/src/${PACKAGE_NAME}/test/pip/role-bindings.yaml"

docker exec "${APISERVER_NAME}" kubectl create clusterrole selfsubjectreview \
	--verb=create --resource=selfsubjectaccessreviews.authorization.k8s.io
