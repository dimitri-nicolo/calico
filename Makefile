PACKAGE_NAME?=github.com/projectcalico/node
GO_BUILD_VER?=v0.32

GIT_USE_SSH = true

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

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
ifdef LOCAL_BUILD
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go:/go/src/github.com/projectcalico/libcalico-go:rw \
	-v $(CURDIR)/../felix:/go/src/github.com/projectcalico/felix:rw \
	-v $(CURDIR)/../typha:/go/src/github.com/projectcalico/typha:rw \
	-v $(CURDIR)/../confd:/go/src/github.com/projectcalico/confd:rw \
	-v $(CURDIR)/../confd:/go/src/github.com/projectcalico/cni-plugin:rw

$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go \
		-replace=github.com/projectcalico/felix=../felix \
		-replace=github.com/projectcalico/typha=../typha \
		-replace=github.com/kelseyhightower/confd=../confd \
		-replace=github.com/projectcalico/cni-plugin=../cni-plugin
endif

include Makefile.common

###############################################################################
CNX_REPOSITORY?=gcr.io/unique-caldron-775/cnx
BUILD_IMAGE?=tigera/cnx-node
PUSH_IMAGES?=$(CNX_REPOSITORY)/tigera/cnx-node
RELEASE_IMAGES?=

# Versions and location of dependencies used in the build.
BIRD_VERSION=v0.3.3-147-g1c33c691
BIRD_IMAGE ?= calico/bird:$(BIRD_VERSION)-$(ARCH)

# Versions and locations of dependencies used in tests.
CALICOCTL_VERSION?=master
CNI_VERSION?=master
TEST_CONTAINER_NAME_VER?=latest
CTL_CONTAINER_NAME?=$(CNX_REPOSITORY)/tigera/calicoctl:$(CALICOCTL_VERSION)-$(ARCH)
TEST_CONTAINER_NAME?=calico/test:$(TEST_CONTAINER_NAME_VER)-$(ARCH)
# If building on amd64 omit the arch in the container name.  Fixme!
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)
ifneq ($(BUILDARCH),amd64)
	ETCD_IMAGE=$(ETCD_IMAGE)-$(ARCH)
endif

HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION)
TEST_CONTAINER_FILES=$(shell find tests/ -type f ! -name '*.created')

# Variables controlling the image
NODE_CONTAINER_CREATED=.calico_node.created-$(ARCH)
NODE_CONTAINER_BIN_DIR=./dist/bin/
NODE_CONTAINER_BINARY = $(NODE_CONTAINER_BIN_DIR)/calico-node-$(ARCH)
WINDOWS_BINARY = $(NODE_CONTAINER_BIN_DIR)/tigera-calico.exe

# Variables for the Windows packaging.
# Name of the Windows release ZIP archive.
WINDOWS_ARCHIVE_ROOT := windows-packaging/TigeraCalico
WINDOWS_ARCHIVE_BINARY := $(WINDOWS_ARCHIVE_ROOT)/tigera-calico.exe
WINDOWS_ARCHIVE_TAG?=$(CNX_GIT_VER)
WINDOWS_ARCHIVE := dist/tigera-calico-windows-$(WINDOWS_ARCHIVE_TAG).zip
# Version of NSSM to download.
WINDOWS_NSSM_VERSION=2.24
# Explicit list of files that we copy in from the vendor directory.  This is required because
# the copying rules we use are pattern-based and they only work with an explicit rule of the
# form "$(WINDOWS_VENDORED_FILES): vendor" (otherwise, make has no way to know that the vendor
# target produces the files we need).
WINDOWS_VENDORED_FILES := \
    vendor/github.com/tigera/confd-private/windows-packaging/config-bgp.ps1 \
    vendor/github.com/tigera/confd-private/windows-packaging/config-bgp.psm1 \
    vendor/github.com/tigera/confd-private/windows-packaging/conf.d/blocks.toml \
    vendor/github.com/tigera/confd-private/windows-packaging/conf.d/peerings.toml \
    vendor/github.com/tigera/confd-private/windows-packaging/templates/blocks.ps1.template \
    vendor/github.com/tigera/confd-private/windows-packaging/templates/peerings.ps1.template \
    vendor/github.com/tigera/confd-private/windows-packaging/config-bgp.ps1 \
    vendor/github.com/tigera/confd-private/windows-packaging/config-bgp.psm1 \
    vendor/github.com/Microsoft/SDN/Kubernetes/windows/hns.psm1 \
    vendor/github.com/Microsoft/SDN/License.txt
# Files to include in the Windows ZIP archive.  We need to list some of these explicitly
# because we need to force them to be built/copied into place.
WINDOWS_ARCHIVE_FILES := \
    $(WINDOWS_ARCHIVE_BINARY) \
    $(WINDOWS_ARCHIVE_ROOT)/README.txt \
    $(WINDOWS_ARCHIVE_ROOT)/*.ps1 \
    $(WINDOWS_ARCHIVE_ROOT)/node/node-service.ps1 \
    $(WINDOWS_ARCHIVE_ROOT)/felix/felix-service.ps1 \
    $(WINDOWS_ARCHIVE_ROOT)/confd/confd-service.ps1 \
    $(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp.ps1 \
    $(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp.psm1 \
    $(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/blocks.toml \
    $(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/peerings.toml \
    $(WINDOWS_ARCHIVE_ROOT)/confd/templates/blocks.ps1.template \
    $(WINDOWS_ARCHIVE_ROOT)/confd/templates/peerings.ps1.template \
    $(WINDOWS_ARCHIVE_ROOT)/cni/calico.exe \
    $(WINDOWS_ARCHIVE_ROOT)/cni/calico-ipam.exe \
    $(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1 \
    $(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt \
    $(WINDOWS_ARCHIVE_ROOT)/libs/calico/calico.psm1

# Variables used by the tests
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')
ST_TO_RUN?=tests/st/
K8ST_TO_RUN?=tests/
# Can exclude the slower tests with "-a '!slow'"
ST_OPTIONS?=

# Variables for building the local binaries that go into the image
NODE_CONTAINER_FILES=$(shell find ./filesystem -type f)

LDFLAGS=-ldflags "\
	-X $(PACKAGE_NAME)/pkg/startup.CNXVERSION=$(CNX_GIT_VER) -X $(PACKAGE_NAME)/pkg/startup.CALICOVERSION=$(CALICO_GIT_VER) \
	-X main.VERSION=$(CALICO_GIT_VER) \
	-X $(PACKAGE_NAME)/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/buildinfo.BuildDate=$(DATE) \
	-X $(PACKAGE_NAME)/buildinfo.GitRevision=$(GIT_COMMIT)"

SRC_FILES=$(shell find ./pkg -name '*.go')

## Clean enough that a new release build will be clean
clean:
	find . -name '*.created' -exec rm -f {} +
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf .go-pkg-cache
	rm -rf certs *.tar $(NODE_CONTAINER_BIN_DIR)
	rm -f $(WINDOWS_ARCHIVE_BINARY) $(WINDOWS_BINARY)
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/templates/*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1
	rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt
	rm -rf dist vendor crds.yaml
	rm -rf filesystem/etc/calico/confd/conf.d filesystem/etc/calico/confd/config filesystem/etc/calico/confd/templates
	# Delete images that we built in this repo
	docker rmi $(BUILD_IMAGE):latest-$(ARCH) || true
	docker rmi $(TEST_CONTAINER_NAME) || true

###############################################################################
# Updating pins
###############################################################################
LIBCALICO_REPO=github.com/tigera/libcalico-go-private
CONFD_REPO=github.com/tigera/confd-private
FELIX_REPO=github.com/tigera/felix-private
TYPHA_REPO=github.com/tigera/typha-private
CNI_PLUGIN_REPO=github.com/tigera/cni-plugin-private

update-pins: replace-libcalico-pin update-confd-pin replace-felix-pin replace-typha-pin replace-cni-pin

###############################################################################
# Building the binary
###############################################################################
build:  $(NODE_CONTAINER_BINARY)

.PHONY: remote-deps
remote-deps:
	mkdir -p filesystem/etc/calico/confd vendor/github.com/tigera vendor/github.com/Microsoft
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go mod download; \
		cp `go list -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/test/crds.yaml crds.yaml; \
		cp -r `go list -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/conf.d filesystem/etc/calico/confd/; \
		cp -r `go list -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/config filesystem/etc/calico/confd/config; \
		cp -r `go list -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/templates filesystem/etc/calico/confd/templates; \
		cp -r `go list -m -f "{{.Dir}}" github.com/kelseyhightower/confd` vendor/github.com/tigera/confd-private; \
		cp -r `go list -m -f "{{.Dir}}" github.com/Microsoft/SDN` vendor/github.com/Microsoft/SDN; \
		chmod -R +w filesystem/etc/calico/confd/ crds.yaml vendor'

$(NODE_CONTAINER_BINARY): $(LOCAL_BUILD_DEP) $(SRC_FILES)
	mkdir -p .go-pkg-cache $(GOMOD_CACHE)
	docker run --rm \
		$(EXTRA_DOCKER_ARGS) \
		-e GOARCH=$(ARCH) \
		-e GOOS=linux \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
		-e GOCACHE=/go-cache \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(BUILD_FLAGS) $(LDFLAGS) ./cmd/calico-node/main.go'

$(WINDOWS_BINARY):
	$(DOCKER_RUN) \
		-e GOOS=windows \
		$(LOCAL_BUILD_MOUNTS) \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(LDFLAGS) ./cmd/calico-node/main.go'

$(WINDOWS_ARCHIVE_ROOT)/cni/calico.exe:
	$(DOCKER_RUN) \
		-e GOOS=windows \
		$(LOCAL_BUILD_MOUNTS) \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(LDFLAGS) ./cmd/calico'

$(WINDOWS_ARCHIVE_ROOT)/cni/calico-ipam.exe:
	$(DOCKER_RUN) \
		-e GOOS=windows \
		$(LOCAL_BUILD_MOUNTS) \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(LDFLAGS) ./cmd/calico-ipam'

###############################################################################
# Building the image
###############################################################################
## Create the image for the current ARCH
image: remote-deps $(BUILD_IMAGE)
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(BUILD_IMAGE): $(NODE_CONTAINER_CREATED)
$(NODE_CONTAINER_CREATED): register ./Dockerfile.$(ARCH) $(NODE_CONTAINER_FILES) $(NODE_CONTAINER_BINARY) remote-deps
ifeq ($(LOCAL_BUILD),true)
	# If doing a local build, copy in local confd templates in case there are changes.
	rm -rf filesystem/etc/calico/confd/templates
	cp -r ../confd/etc/calico/confd/templates filesystem/etc/calico/confd/templates
endif
	# Check versions of the binaries that we're going to use to build the image.
	# Since the binaries are built for Linux, run them in a container to allow the
	# make target to be run on different platforms (e.g. MacOS).
	docker run --rm -v $(CURDIR)/dist/bin:/go/bin:rw $(CALICO_BUILD) /bin/sh -c "\
	  echo; echo calico-node-$(ARCH) -v;	 /go/bin/calico-node-$(ARCH) -v; \
	"
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) . --build-arg BIRD_IMAGE=$(BIRD_IMAGE) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f ./Dockerfile.$(ARCH)
	touch $@

###############################################################################
# FV Tests
###############################################################################
## Run the ginkgo FVs
fv: run-k8s-apiserver
	 $(DOCKER_RUN) -e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		ginkgo -cover -r -skipPackage vendor pkg/startup pkg/allocateip $(GINKGO_ARGS)'

# etcd is used by the STs
.PHONY: run-etcd
run-etcd:
	@-docker rm -f calico-etcd
	docker run --detach \
	--net=host \
	--name calico-etcd $(ETCD_IMAGE) \
	etcd \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379" \
	--listen-client-urls "http://0.0.0.0:2379"

# Kubernetes apiserver used for tests
run-k8s-apiserver: remote-deps stop-k8s-apiserver run-etcd
	docker run \
		--net=host --name st-apiserver \
		-v $(CURDIR):/manifests \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		--detach \
		${HYPERKUBE_IMAGE} sh -c '\
		go mod download; \
		/hyperkube apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--logtostderr=true'

	# Wait until we can configure a cluster role binding which allows anonymous auth.
	while ! docker exec st-apiserver kubectl create \
		clusterrolebinding anonymous-admin \
		--clusterrole=cluster-admin \
		--user=system:anonymous 2>/dev/null ; \
		do echo "Waiting for st-apiserver to come up"; \
		sleep 1; \
		done

	# ClusterRoleBinding created

	# Create CustomResourceDefinition (CRD) for Calico resources
	# from the manifest crds.yaml
	while ! docker exec st-apiserver kubectl \
		apply -f /manifests/crds.yaml; \
		do echo "Trying to create CRDs"; \
		sleep 1; \
		done

# Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f st-apiserver

ut:
	@echo "No UTs available"

###############################################################################
# System tests
# - Support for running etcd (both securely and insecurely)
###############################################################################
# Pull calicoctl and CNI plugin binaries with versions as per XXX_VER
# variables.  These are used for the STs.
dist/calicoctl:
	-docker rm -f calicoctl
	docker pull $(CTL_CONTAINER_NAME)
	docker create --name calicoctl $(CTL_CONTAINER_NAME)
	docker cp calicoctl:calicoctl dist/calicoctl && \
	  test -e dist/calicoctl && \
	  touch dist/calicoctl
	-docker rm -f calicoctl

dist/calico-cni-plugin dist/calico-ipam-plugin:
	-docker rm -f calico-cni
	docker pull $(CNX_REPOSITORY)/tigera/cni:$(CNI_VERSION)
	docker create --name calico-cni $(CNX_REPOSITORY)/tigera/cni:$(CNI_VERSION)
	docker cp calico-cni:/opt/cni/bin/calico dist/calico-cni-plugin && \
	  test -e dist/calico-cni-plugin && \
	  touch dist/calico-cni-plugin
	docker cp calico-cni:/opt/cni/bin/calico-ipam dist/calico-ipam-plugin && \
	  test -e dist/calico-ipam-plugin && \
	  touch dist/calico-ipam-plugin
	-docker rm -f calico-cni

# Create images for containers used in the tests
busybox.tar:
	docker pull $(ARCH)/busybox:latest
	docker save --output busybox.tar $(ARCH)/busybox:latest

workload.tar:
	cd workload && docker build -t workload --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
	docker save --output workload.tar workload

stop-etcd:
	@-docker rm -f calico-etcd

IPT_ALLOW_ETCD:=-A INPUT -i docker0 -p tcp --dport 2379 -m comment --comment "calico-st-allow-etcd" -j ACCEPT

# Create the calico/test image
test_image: calico_test.created
calico_test.created: $(TEST_CONTAINER_FILES)
	cd calico_test && docker build --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH).calico_test -t $(TEST_CONTAINER_NAME) .
	touch calico_test.created

cnx-node.tar: $(NODE_CONTAINER_CREATED)
	# Check versions of the Calico binaries that will be in cnx-node.tar.
	# Since the binaries are built for Linux, run them in a container to allow the
	# make target to be run on different platforms (e.g. MacOS).
	docker run --rm $(BUILD_IMAGE):latest-$(ARCH) /bin/sh -c "\
	  echo bird --version;	 /bin/bird --version; \
	"
	docker save --output $@ $(BUILD_IMAGE):latest-$(ARCH)

.PHONY: st-checks
st-checks:
	# Check that we're running as root.
	test `id -u` -eq '0' || { echo "STs must be run as root to allow writes to /proc"; false; }

	# Insert an iptables rule to allow access from our test containers to etcd
	# running on the host.
	iptables-save | grep -q 'calico-st-allow-etcd' || iptables $(IPT_ALLOW_ETCD)

.PHONY: dual-tor-test
dual-tor-test: cnx-node.tar calico_test.created
	$(MAKE) dual-tor-setup
	$(MAKE) dual-tor-run-test
	$(MAKE) dual-tor-cleanup

.PHONY: dual-tor-setup
dual-tor-setup: cnx-node.tar calico_test.created
	git submodule update --init
	cd tests/kind && make
	curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.3/bin/linux/amd64/kubectl
	chmod +x ./kubectl
	GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) STEPS=setup tests/k8st/dual-tor/dualtor.sh

.PHONY: dual-tor-run-test
dual-tor-run-test:
	docker run -t --rm \
	    -v $(PWD):/code \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v ${HOME}/.kube/kind-config-kind:/root/.kube/config \
	    -v $(PWD)/kubectl:/root/bin/kubectl \
	    --privileged \
	    --net host \
	${TEST_CONTAINER_NAME} \
	    sh -c 'echo "container started.." && cp /root/bin/kubectl /bin/kubectl && echo "kubectl copied." && \
	     cd /code/tests/k8st &&  nosetests dual-tor-tests/test_dual_tor.py -s --nocapture --nologcapture -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

.PHONY: dual-tor-cleanup
dual-tor-cleanup:
	STEPS=cleanup tests/k8st/dual-tor/dualtor.sh
	rm ./kubectl

## Get the kubeadm-dind-cluster script
K8ST_VERSION?=v1.12
DIND_SCR?=dind-cluster-$(K8ST_VERSION).sh
GCR_IO_PULL_SECRET?=${HOME}/.docker/config.json
TSEE_TEST_LICENSE?=${HOME}/new-test-customer-license.yaml

.PHONY: k8s-test
## Run the k8s tests
k8s-test: cnx-node.tar $(NODE_CONTAINER_CREATED)
	$(MAKE) k8s-stop
	$(MAKE) k8s-start
	$(MAKE) k8s-check-setup
	$(MAKE) k8s-run-test
	#$(MAKE) k8s-stop

.PHONY: k8s-start
## Start k8s cluster
k8s-start: $(NODE_CONTAINER_CREATED) tests/k8st/$(DIND_SCR)
	CNI_PLUGIN=calico \
	CALICO_VERSION=master \
	CALICO_NODE_IMAGE=$(BUILD_IMAGE):latest-$(ARCH) \
	POD_NETWORK_CIDR=192.168.0.0/16 \
	SKIP_SNAPSHOT=y \
	GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) \
	TSEE_TEST_LICENSE=$(TSEE_TEST_LICENSE) \
	tests/k8st/$(DIND_SCR) up

.PHONY: k8s-check-setup
k8s-check-setup:
	ls -l ${HOME}/.kubeadm-dind-cluster/
	${HOME}/.kubeadm-dind-cluster/kubectl get no -o wide
	${HOME}/.kubeadm-dind-cluster/kubectl get po -o wide --all-namespaces
	${HOME}/.kubeadm-dind-cluster/kubectl get svc -o wide --all-namespaces
	${HOME}/.kubeadm-dind-cluster/kubectl get deployments -o wide --all-namespaces
	${HOME}/.kubeadm-dind-cluster/kubectl get ds -o wide --all-namespaces

.PHONY: k8s-stop
## Stop k8s cluster
k8s-stop: tests/k8st/$(DIND_SCR)
	tests/k8st/$(DIND_SCR) down
	tests/k8st/$(DIND_SCR) clean

.PHONY: k8s-run-test
## Run k8st in an existing k8s cluster
##
## Note: if you're developing and want to see test output as it
## happens, instead of only later and if the test fails, add "-s
## --nocapture --nologcapture" to K8ST_TO_RUN.  For example:
##
## make k8s-test K8ST_TO_RUN="tests/test_dns_policy.py -s --nocapture --nologcapture"
k8s-run-test: calico_test.created
## Only execute remove-go-build-image if flag is set
ifeq ($(REMOVE_GOBUILD_IMG),true)
	$(MAKE) remove-go-build-image
endif
	docker run -t \
	    -v $(CURDIR):/code \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v /home/$(USER)/.kube/config:/root/.kube/config \
	    -v /home/$(USER)/.kubeadm-dind-cluster:/root/.kubeadm-dind-cluster \
	    --privileged \
	    --net host \
	$(TEST_CONTAINER_NAME) \
	    sh -c 'cp /root/.kubeadm-dind-cluster/kubectl /bin/kubectl && ls -ltr /bin/kubectl && which kubectl && cd /code/tests/k8st && \
		   nosetests $(K8ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

# Needed for Semaphore CI (where disk space is a real issue during k8s-test)
.PHONY: remove-go-build-image
remove-go-build-image:
	@echo "Removing $(CALICO_BUILD) image to save space needed for testing ..."
	@-docker rmi $(CALICO_BUILD)

.PHONY: st
## Run the system tests
st: remote-deps dist/calicoctl busybox.tar cnx-node.tar workload.tar run-etcd calico_test.created dist/calico-cni-plugin dist/calico-ipam-plugin
	# Check versions of Calico binaries that ST execution will use.
	docker run --rm -v $(CURDIR)/dist:/go/bin:rw $(CALICO_BUILD) /bin/sh -c "\
	  echo; echo calicoctl version;	  /go/bin/calicoctl version; \
	  echo; echo calico-cni-plugin -v;       /go/bin/calico-cni-plugin -v; \
	  echo; echo calico-ipam-plugin -v;      /go/bin/calico-ipam-plugin -v; echo; \
	"
	# Use the host, PID and network namespaces from the host.
	# Privileged is needed since 'calico node' write to /proc (to enable ip_forwarding)
	# Map the docker socket in so docker can be used from inside the container
	# HOST_CHECKOUT_DIR is used for volume mounts on containers started by this one.
	# All of code under test is mounted into the container.
	#   - This also provides access to calicoctl and the docker client
	# $(MAKE) st-checks
	docker run --uts=host \
		   --pid=host \
		   --net=host \
		   --privileged \
		   -v $(CURDIR):/code \
		   -e HOST_CHECKOUT_DIR=$(CURDIR) \
		   -e DEBUG_FAILURES=$(DEBUG_FAILURES) \
		   -e MY_IP=$(LOCAL_IP_ENV) \
		   -e NODE_CONTAINER_NAME=$(BUILD_IMAGE):latest-$(ARCH) \
		   --rm -t \
		   -v /var/run/docker.sock:/var/run/docker.sock \
		   $(TEST_CONTAINER_NAME) \
		   sh -c 'nosetests $(ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/nosetests.xml" --with-timer $(ST_OPTIONS)'
	$(MAKE) stop-etcd

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean mod-download static-checks ut fv image-all build-windows-archive st

## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=${BRANCH_NAME} EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) CALICO_GIT_VER=$(CALICO_GIT_VER_RELEASE) VERSION=$(VERSION) release-tag
	$(MAKE) CALICO_GIT_VER=$(CALICO_GIT_VER_RELEASE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) tag-base-images-all
	$(MAKE) CALICO_GIT_VER=$(CALICO_GIT_VER_RELEASE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make CALICO_GIT_VER=$(CALICO_GIT_VER_RELEASE) VERSION=$(VERSION) release-publish"
	@echo ""

## Produces a git tag for the release.
release-tag: release-prereqs release-notes
	git tag $(VERSION) -F release-notes-$(VERSION)
	@echo ""
	@echo "Now you can build the release:"
	@echo ""
	@echo "  make VERSION=$(VERSION) release-build"
	@echo ""

## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif
	$(MAKE) image-all
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=$(VERSION)
	# Generate the `latest` images.
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=latest

## Produces the Windows ZIP archive for the release.
release-windows-archive $(WINDOWS_ARCHIVE): release-prereqs
	$(MAKE) build-windows-archive WINDOWS_ARCHIVE_TAG=$(VERSION)

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	if ! docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) versions | grep '^$(VERSION)$$'; then echo "Reported version:" `docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) versions` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	echo "" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	# Push images.
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=$(VERSION)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "If this is the latest stable release, then run the following to push 'latest' images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish-latest"
	@echo ""

# WARNING: Only run this target if this release is the latest stable release. Do NOT
# run this target for alpha / beta / release candidate builds, or patches to earlier Calico versions.
## Pushes `latest` release images. WARNING: Only run this for latest stable releases.
release-publish-latest: release-verify
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=latest

.PHONY: node-test-at
# Run docker-image acceptance tests
node-test-at: release-prereqs
	docker run -v $(PWD)/tests/at/calico_node_goss.yaml:/tmp/goss.yaml \
	  $(BUILD_IMAGE):$(VERSION) /bin/sh -c ' \
	   apk --no-cache add wget ca-certificates && \
	   wget -q -O /tmp/goss https://github.com/aelsabbahy/goss/releases/download/v0.3.4/goss-linux-amd64 && \
	   chmod +rx /tmp/goss && \
	   /tmp/goss --gossfile /tmp/goss.yaml validate'

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif
ifndef CALICO_GIT_VER_RELEASE
	$(error CALICO_GIT_VER_RELEASE is undefined - run using make release CALICO_GIT_VER_RELEASE=vX.Y.Z)
endif

###############################################################################
# Image build/push
###############################################################################
# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))

sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

## push all arches
push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using
	# $(DOCKER_CONFIG) holds the docker login credentials path to credentials based on
	# manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*:$(IMAGETAG))-ARCH --target $(call unescapefs,$*:$(IMAGETAG))"

## push default amd64 arch where multi-arch manifest is not supported
push-non-manifests: imagetag $(addprefix sub-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))
sub-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker push $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of one arch for all supported registries
tag-images: imagetag $(addprefix sub-single-tag-images-arch-,$(call escapefs,$(PUSH_IMAGES))) $(addprefix sub-single-tag-images-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))

sub-single-tag-images-arch-%:
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(VALIDARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

###############################################################################
# Utilities
###############################################################################
$(info "Build dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "BIRD_VERSION" $(BIRD_VERSION)))

$(info "Test dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "CNI_VERSION" $(CNI_VERSION)))

$(info "Calico git version")
$(info $(shell printf "%-21s = %-10s\n" "GIT_VERSION" $(GIT_VERSION)))
