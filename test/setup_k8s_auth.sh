#!/bin/bash -e

docker exec st-apiserver kubectl apply -f /test/role.yaml
docker exec st-apiserver kubectl apply -f /test/token-binding.yaml

docker exec st-apiserver kubectl apply -f /test/pip/roles.yaml
docker exec st-apiserver kubectl apply -f /test/pip/role-bindings.yaml

docker exec st-apiserver kubectl create clusterrole selfsubjectreview \
	--verb=create --resource=selfsubjectaccessreviews.authorization.k8s.io
