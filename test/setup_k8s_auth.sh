#!/bin/bash -e

docker exec st-apiserver kubectl apply -f /test/role.yaml
docker exec st-apiserver kubectl apply -f /test/basic-binding.yaml
docker exec st-apiserver kubectl apply -f /test/token-binding.yaml

docker exec st-apiserver kubectl apply -f /test/pip/roles.yaml
docker exec st-apiserver kubectl apply -f /test/pip/role-bindings.yaml

docker exec st-apiserver kubectl create clusterrole selfsubjectreview \
	--verb=create --resource=selfsubjectaccessreviews.authorization.k8s.io

docker exec st-apiserver kubectl create clusterrolebinding basicuserall \
  --clusterrole=selfsubjectreview --user=basicuserall
docker exec st-apiserver kubectl create clusterrolebinding basicuserflowonlyuid \
  --clusterrole=selfsubjectreview --user=basicuserflowonlyuid
docker exec st-apiserver kubectl create clusterrolebinding basicuserauditonly \
  --clusterrole=selfsubjectreview --user=basicuserauditonly
docker exec st-apiserver kubectl create clusterrolebinding basicusernone \
  --clusterrole=selfsubjectreview --user=basicusernone
docker exec st-apiserver kubectl create clusterrolebinding basicuserallgrp \
  --clusterrole=selfsubjectreview --user=basicuserallgrp
# don't do above for basicusernoselfaccess


#Policy Impact test users
docker exec st-apiserver kubectl create clusterrolebinding basicpolicyreadonly \
  --clusterrole=selfsubjectreview --user=basicpolicyreadonly

docker exec st-apiserver kubectl create clusterrolebinding basicpolicycrud \
  --clusterrole=selfsubjectreview --user=basicpolicycrud

docker exec st-apiserver kubectl create clusterrolebinding basicpolicyreadcreate \
  --clusterrole=selfsubjectreview --user=basicpolicyreadcreate

docker exec st-apiserver kubectl create clusterrolebinding basicpolicyreadupdate \
  --clusterrole=selfsubjectreview --user=basicpolicyreadupdate

docker exec st-apiserver kubectl create clusterrolebinding basicpolicyreaddelete \
  --clusterrole=selfsubjectreview --user=basicpolicyreaddelete
