#!/bin/bash -e

docker exec st-apiserver kubectl apply -f /test/crds.yaml
docker exec st-apiserver kubectl apply -f /test/rbac/reports.yaml
docker exec st-apiserver kubectl apply -f /test/rbac/role.yaml
docker exec st-apiserver kubectl apply -f /test/rbac/role-bindings.yaml

docker exec st-apiserver kubectl create clusterrole selfsubjectreview \
	--verb=create --resource=selfsubjectaccessreviews.authorization.k8s.io

docker exec st-apiserver kubectl create clusterrolebinding basicuserallsar \
  --clusterrole=selfsubjectreview --user=basicuserall
docker exec st-apiserver kubectl create clusterrolebinding basicuserlimitedbsar \
  --clusterrole=selfsubjectreview --user=basicuserlimited
docker exec st-apiserver kubectl create clusterrolebinding basicusernoauditsar \
  --clusterrole=selfsubjectreview --user=basicusernoaudit

