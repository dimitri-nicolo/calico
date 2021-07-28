PACKAGE_NAME=github.com/projectcalico/calicoctl
GO_BUILD_VER=v0.53

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_CALICOCTL_PRIVATE_PROJECT_ID)

SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_TS_QUERYSERVER_PROJECT_ID)

GIT_USE_SSH = true

KUBE_APISERVER_PORT?=8080
KUBE_MOCK_NODE_MANIFEST?=mock-node.yaml

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
ifdef LOCAL_BUILD
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go:/go/src/github.com/projectcalico/libcalico-go:rw
$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
endif

CALICOCTL_IMAGE       ?=tigera/calicoctl
BUILD_IMAGES          ?=$(CALICOCTL_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# Remove any excluded architectures since for calicoctl we want to build everything.
EXCLUDEARCH?=

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

###############################################################################

CALICOCTL_DIR=calicoctl
CTL_CONTAINER_CREATED=$(CALICOCTL_DIR)/.calico_ctl.created-$(ARCH)
SRC_FILES=$(shell find $(CALICOCTL_DIR) -name '*.go')

TEST_CONTAINER_NAME ?= calico/test

CALICOCTL_GIT_REVISION?=$(shell git rev-parse --short HEAD)

LDFLAGS=-ldflags "-X $(PACKAGE_NAME)/v3/calicoctl/commands.VERSION=$(GIT_VERSION) \
	-X $(PACKAGE_NAME)/v3/calicoctl/commands.GIT_REVISION=$(CALICOCTL_GIT_REVISION) -s -w"

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} \;
	rm -rf .go-pkg-cache bin build certs *.tar vendor Makefile.common* calicoctl/commands/report
	docker rmi $(CALICOCTL_IMAGE):latest-$(ARCH) || true
	docker rmi $(CALICOCTL_IMAGE):$(VERSION)-$(ARCH) || true
ifeq ($(ARCH),amd64)
	docker rmi $(CALICOCTL_IMAGE):latest || true
	docker rmi $(CALICOCTL_IMAGE):$(VERSION) || true
endif

###############################################################################
# Updating pins
###############################################################################
LICENSING_BRANCH=$(PIN_BRANCH)
LICENSING_REPO=github.com/tigera/licensing
LIBCALICO_REPO=github.com/tigera/libcalico-go-private

update-licensing-pin:
	$(call update_pin,github.com/tigera/licensing,$(LICENSING_REPO),$(LICENSING_BRANCH))

update-pins:  update-licensing-pin update-api-pin replace-libcalico-pin

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/calicoctl-linux-,$(VALIDARCHES)) bin/calicoctl-windows-amd64.exe bin/calicoctl-darwin-amd64
.PHONY: build
## Build the binary for the current architecture and platform
build: bin/calicoctl-$(BUILDOS)-$(ARCH)
# The supported different binary names. For each, ensure that an OS and ARCH is set
bin/calicoctl-%-amd64: ARCH=amd64
bin/calicoctl-%-armv7: ARCH=armv7
bin/calicoctl-%-arm64: ARCH=arm64
bin/calicoctl-%-ppc64le: ARCH=ppc64le
bin/calicoctl-%-s390x: ARCH=s390x
bin/calicoctl-darwin-amd64: BUILDOS=darwin
bin/calicoctl-windows-amd64: BUILDOS=windows
bin/calicoctl-linux-%: BUILDOS=linux
# We reinvoke make here to re-evaluate BUILDOS and ARCH so the correct values
# for multi-platform builds are used. When make is initially invoked, BUILDOS
# and ARCH are defined with default values (Linux and amd64).
bin/calicoctl-%: $(LOCAL_BUILD_DEP) $(SRC_FILES)
	$(MAKE) build-calicoctl BUILDOS=$(BUILDOS) ARCH=$(ARCH)
build-calicoctl:
	mkdir -p bin
	$(DOCKER_RUN) $(EXTRA_DOCKER_ARGS) \
	  -e CALICOCTL_GIT_REVISION=$(CALICOCTL_GIT_REVISION) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build -v -o bin/calicoctl-$(BUILDOS)-$(ARCH) $(LDFLAGS) "./calicoctl/calicoctl.go"'

# Overrides for the binaries that need different output names
bin/calicoctl: bin/calicoctl-linux-amd64
	cp $< $@

bin/calicoctl-windows-amd64.exe: bin/calicoctl-windows-amd64
	mv $< $@

gen-crds: remote-deps
	$(DOCKER_RUN) \
	  -v $(CURDIR)/calicoctl/commands/crds:/go/src/$(PACKAGE_NAME)/calicoctl/commands/crds \
	  $(CALICO_BUILD) \
	  sh -c 'cd /go/src/$(PACKAGE_NAME)/calicoctl/commands/crds && go generate'

remote-deps: mod-download	
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
		$(GIT_CONFIG_SSH) \
		cp -r `go list -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/config .; \
		chmod -R +w config/'

###############################################################################
# Building the image
###############################################################################
.PHONY: image $(CALICOCTL_IMAGE)
image: $(CALICOCTL_IMAGE)
$(CALICOCTL_IMAGE): $(CTL_CONTAINER_CREATED)
$(CTL_CONTAINER_CREATED): Dockerfile.$(ARCH) bin/calicoctl-linux-$(ARCH)
	docker build -t $(CALICOCTL_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(CALICOCTL_IMAGE):latest-$(ARCH) $(CALICOCTL_IMAGE):latest
endif
	touch $@

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev.
ut: $(LOCAL_BUILD_DEP) bin/calicoctl-linux-amd64
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'cd /go/src/$(PACKAGE_NAME) && ginkgo -cover -r calicoctl/*'

###############################################################################
# FVs
###############################################################################
.PHONY: fv
## Run the tests in a container. Useful for CI, Mac dev.
fv: $(LOCAL_BUILD_DEP) bin/calicoctl-linux-amd64
	$(MAKE) run-etcd-host
	# We start two API servers in order to test multiple kubeconfig support
	$(MAKE) run-kubernetes-master KUBE_APISERVER_PORT=8080 KUBE_MOCK_NODE_MANIFEST=mock-node.yaml
	$(MAKE) run-kubernetes-master KUBE_APISERVER_PORT=8082 KUBE_MOCK_NODE_MANIFEST=mock-node-second.yaml
	# Run the tests
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) cd /go/src/$(PACKAGE_NAME) && go test ./tests/fv'
	# Cleanup
	$(MAKE) stop-etcd
	$(MAKE) stop-kubernetes-master KUBE_APISERVER_PORT=8080
	$(MAKE) stop-kubernetes-master KUBE_APISERVER_PORT=8082

###############################################################################
# STs
###############################################################################
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')
# To run a specific test, set ST_TO_RUN to testfile.py:class.method
# e.g. ST_TO_RUN="tests/st/calicoctl/test_crud.py:TestCalicoctlCommands.test_get_delete_multiple_names"
ST_TO_RUN?=tests/st/calicoctl/
# Can exclude the slower tests with "-a '!slow'"
ST_OPTIONS?=

.PHONY: st
## Run the STs in a container
st: bin/calicoctl-linux-amd64
	$(MAKE) run-etcd-host
	$(MAKE) run-kubernetes-master
	# Use the host, PID and network namespaces from the host.
	# Privileged is needed since 'calico node' write to /proc (to enable ip_forwarding)
	# Map the docker socket in so docker can be used from inside the container
	# All of code under test is mounted into the container.
	#   - This also provides access to calicoctl and the docker client
	docker run --net=host --privileged \
		   -e MY_IP=$(LOCAL_IP_ENV) \
		   --rm -t \
		   -v $(CURDIR):/code \
		   -v /var/run/docker.sock:/var/run/docker.sock \
		   $(TEST_CONTAINER_NAME) \
		   sh -c 'nosetests $(ST_TO_RUN) -sv --nologcapture  --with-xunit --xunit-file="/code/report/nosetests.xml" --with-timer $(ST_OPTIONS)'
	$(MAKE) stop-etcd
	$(MAKE) stop-kubernetes-master

## Etcd is used by the STs
# NOTE: https://quay.io/repository/coreos/etcd is available *only* for the following archs with the following tags:
# amd64: 3.3.7
# arm64: 3.3.7-arm64
# ppc64le: 3.3.7-ppc64le
# s390x is not available
# armv7 is not available
COREOS_ETCD?=quay.io/coreos/etcd:$(ETCD_VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
COREOS_ETCD=quay.io/coreos/etcd:$(ETCD_VERSION)
endif
.PHONY: run-etcd-host
run-etcd-host:
	@-docker rm -f calico-etcd
	docker run --detach \
	--net=host \
	--name calico-etcd \
	$(COREOS_ETCD) \
	etcd \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379" \
	--listen-client-urls "http://0.0.0.0:2379"

.PHONY: stop-etcd
stop-etcd:
	@-docker rm -f calico-etcd

## Run a local kubernetes master with API via hyperkube
run-kubernetes-master: stop-kubernetes-master remote-deps
	# Run a Kubernetes apiserver using Docker.
	docker run \
		--net=host --name st-apiserver-${KUBE_APISERVER_PORT} \
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} kube-apiserver \
			--bind-address=0.0.0.0 \
			--secure-port=1${KUBE_APISERVER_PORT} \
			--insecure-bind-address=0.0.0.0 \
			--port=${KUBE_APISERVER_PORT} \
	        	--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--logtostderr=true

	# Wait until the apiserver is accepting requests.
	while ! docker exec st-apiserver-${KUBE_APISERVER_PORT} kubectl get nodes; do echo "Waiting for apiserver to come up..."; sleep 2; done

	# And run the controller manager.
	docker run \
		--net=host --name st-controller-manager-${KUBE_APISERVER_PORT} \
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} kube-controller-manager \
                        --master=127.0.0.1:${KUBE_APISERVER_PORT} \
                        --min-resync-period=3m \
                        --allocate-node-cidrs=true \
                        --cluster-cidr=10.10.0.0/16 \
                        --v=5

	# Create a Node in the API for the tests to use.
	while ! docker run \
	    --net=host \
	    --rm \
		-v $(CURDIR):/manifests \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} kubectl \
		--server=http://127.0.0.1:${KUBE_APISERVER_PORT} \
		apply -f /manifests/tests/st/manifests/${KUBE_MOCK_NODE_MANIFEST}; \
		do echo "Waiting for node to apply successfully..."; sleep 2; done

	# Create a namespace in the API for the tests to use.
	-docker run \
	    --net=host \
	    --rm \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} kubectl \
		--server=http://127.0.0.1:${KUBE_APISERVER_PORT} \
		apply -f /manifests/tests/st/manifests/mock-node.yaml

	# Apply Calico CRDs for tests that use KDD mode.
	docker run \
	    --net=host \
	    --rm \
		-v  $(CURDIR):/manifests \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} kubectl \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/config/crd/
	
## Stop the local kubernetes master
stop-kubernetes-master:
	# Delete the cluster role binding.
	-docker exec st-apiserver-${KUBE_APISERVER_PORT} kubectl delete clusterrolebinding anonymous-admin

	# Stop master components.
	-docker rm -f st-apiserver-${KUBE_APISERVER_PORT} st-controller-manager-${KUBE_APISERVER_PORT}

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: mod-download build-all static-checks test

## Avoid unplanned go.sum updates
.PHONY: undo-go-sum check-dirty
undo-go-sum:
	@echo "Undoing go.sum update..."
	git checkout -- go.sum

## Check if generated image is dirty
check-dirty: undo-go-sum
	@if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated image is dirty:"; \
	  git status --porcelain; \
	  false; \
	fi

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: check-dirty image-all cd-common
