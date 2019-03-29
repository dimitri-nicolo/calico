#!/bin/bash -e


docker exec st-apiserver kubectl apply -f /test/role.yaml
docker exec st-apiserver kubectl apply -f /test/basic-binding.yaml
docker exec st-apiserver kubectl apply -f /test/token-binding.yaml

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
