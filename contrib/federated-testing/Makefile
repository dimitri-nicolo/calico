LOCAL_IP_ENV=$(shell ip -o -f inet addr show docker0 | grep -Po 'inet \K[\d.]+')

# This is for running a "local" etcd
run-etcd:
	-docker rm -f calico-etcd
	docker run --detach \
	-p 2379:2379 \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:v3.3.2 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379" \
	--listen-client-urls "http://0.0.0.0:2379"

# These etcds represent remote clusters
run-etcd-1:
	-docker rm -f calico-etcd-remote1
	docker run --detach \
	--entrypoint=/usr/local/bin/etcd \
	-p 2381:2381 \
	--name calico-etcd-remote1 quay.io/coreos/etcd:v3.3.2 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2381" \
	--listen-client-urls "http://0.0.0.0:2381"

run-etcd-2:
	-docker rm -f calico-etcd-remote2
	docker run --detach \
	--entrypoint=/usr/local/bin/etcd \
	-p 2382:2382 \
	--name calico-etcd-remote2 quay.io/coreos/etcd:v3.3.2 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2382" \
	--listen-client-urls "http://0.0.0.0:2382"

run-etcd-3:
	-docker rm -f calico-etcd-remote3
	docker run --detach \
	--entrypoint=/usr/local/bin/etcd \
	-p 2383:2383 \
	--name calico-etcd-remote3 quay.io/coreos/etcd:v3.3.2 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2383" \
	--listen-client-urls "http://0.0.0.0:2383"

# This runs a k8s api server talking to etcd-1
run-k8s-apiserver1: run-etcd-1
	-docker rm -f calico-k8s-apiserver
	docker run --detach -p 8080:8080 \
	  --name calico-k8s-apiserver \
  	gcr.io/google_containers/hyperkube-amd64:v1.10.0 \
		  /hyperkube apiserver --etcd-servers=http://$(LOCAL_IP_ENV):2381 \
		  --service-cluster-ip-range=10.101.0.0/16 --insecure-bind-address 0.0.0.0

# Create a local etcd and 1 remote etcdv3 and 1 remote kdd
test: run-etcd run-etcd-2 run-etcd-1 run-k8s-apiserver1
	while ! kubectl -s http://127.0.0.1:8080 version >/dev/null; do sleep 0.1; done
	# The -s option might help to override any local k8s config that might be present
	kubectl -s http://127.0.0.1:8080 apply -f ../../test/crds.yaml
	./generate_resources.py rcc --num=1 --ds=kubernetes | ETCD_ENDPOINTS=http://127.0.0.1:2379 calicoctl apply -f -
	./generate_resources.py rcc --num=1 --ds=etcdv3 --offset=2 | ETCD_ENDPOINTS=http://127.0.0.1:2379 calicoctl apply -f -
	ETCD_ENDPOINTS=http://127.0.0.1:2379 calicoctl get rcc -o wide

add-data-to-etcd-2:
	./generate_resources.py wep --num 2500 --per-node 100| ETCD_ENDPOINTS=http://127.0.0.1:2382 calicoctl apply -f -
	./generate_resources.py hep --num 2500 | ETCD_ENDPOINTS=http://127.0.0.1:2382 calicoctl apply -f -
	./generate_resources.py profile --num 50 | ETCD_ENDPOINTS=http://127.0.0.1:2382 calicoctl apply -f -

add-data-to-k8s:
	./generate_resources.py hep --num 25 | K8S_API_ENDPOINT=http://127.0.0.1:8080 DATASTORE_TYPE=kubernetes calicoctl apply -f -
	./generate_resources.py profile --num 5 --ds=kubernetes | kubectl -s http://127.0.0.1:8080 apply -f -
	# Fake WEPs can't be created. They can't be created through calicoctl and ones created through kubectl can't have an IP address (so they are useless and filtered out) since it's in the read only status section.