PACKAGE_NAME?=github.com/projectcalico/node
GO_BUILD_VER?=v0.63

GIT_USE_SSH = true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_NODE_PRIVATE_PROJECT_ID)

WINDOWS_VERSIONS?=1809 2004 20H2 ltsc2022

NODE_IMAGE            ?=tigera/cnx-node
WINDOWS_UPGRADE_IMAGE ?=tigera/calico-windows-upgrade

# If this is a windows release build the windows upgrade image.
# ARCHES will contain the values in WINDOWS_VERSIONS, prefixed with "windows-".
ifdef WINDOWS_RELEASE
BUILD_IMAGES          ?=$(WINDOWS_UPGRADE_IMAGE)
ARCHES                ?= $(patsubst %,windows-%,$(WINDOWS_VERSIONS))
else
BUILD_IMAGES          ?=$(NODE_IMAGE)
endif

DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*
LIBBPF_DOCKER_PATH=/go/src/github.com/projectcalico/node/bin/third-party/libbpf/src
BPF_GPL_DOCKER_PATH=/go/src/github.com/projectcalico/node/bin/bpf/bpf-gpl
LIBBPF_PATH=./bin/third-party/libbpf/src

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
ifdef LOCAL_BUILD
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go:/go/src/github.com/projectcalico/libcalico-go:rw \
	-v $(CURDIR)/../felix:/go/src/github.com/projectcalico/felix:rw \
	-v $(CURDIR)/../typha:/go/src/github.com/projectcalico/typha:rw \
	-v $(CURDIR)/../confd:/go/src/github.com/projectcalico/confd:rw \
	-v $(CURDIR)/../cni-plugin:/go/src/github.com/projectcalico/cni-plugin:rw

$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go \
		-replace=github.com/projectcalico/felix=../felix \
		-replace=github.com/projectcalico/typha=../typha \
		-replace=github.com/kelseyhightower/confd=../confd \
		-replace=github.com/projectcalico/cni-plugin=../cni-plugin
endif

# These arches not yet building in node-private
EXCLUDEARCH?=s390x arm64 ppc64le

# This gets embedded into node as the Calico version, the Enterprise release
# is based off of. This should be updated everytime a new opensource Calico
# release is merged into node-private.
CALICO_VERSION=v3.21.0

# Add in local static-checks
LOCAL_CHECKS=check-boring-ssl

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

# Required for eBPF support in ARM64
ifeq ($(ARCH),arm64)
# Forces ARM64 build image to be used in a crosscompilation run.
CALICO_BUILD:=$(CALICO_BUILD)-$(ARCH)
# Prevents docker from tagging the output image incorrectly as amd64.
TARGET_PLATFORM=--platform=linux/arm64/v8
endif

###############################################################################

# Versions and location of dependencies used in the build.
BIRD_VERSION=v0.3.3-184-g202a2186
BIRD_IMAGE ?= calico/bird:$(BIRD_VERSION)-$(ARCH)
BIRD_SOURCE=filesystem/included-source/bird-$(BIRD_VERSION).tar.gz
FELIX_GPL_SOURCE=filesystem/included-source/felix-ebpf-gpl.tar.gz
INCLUDED_SOURCE=$(BIRD_SOURCE) $(FELIX_GPL_SOURCE)

# Versions and locations of dependencies used in tests.
CNX_REPOSITORY          ?=gcr.io/unique-caldron-775/cnx
CALICOCTL_VERSION       ?=master
CNI_VERSION             ?=master
TEST_CONTAINER_NAME_VER ?=latest
CTL_CONTAINER_NAME      ?=$(CNX_REPOSITORY)/tigera/calicoctl:$(CALICOCTL_VERSION)-$(ARCH)
TEST_CONTAINER_NAME     ?=calico/test:$(TEST_CONTAINER_NAME_VER)-$(ARCH)
# If building on amd64 omit the arch in the container name.  Fixme!
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)
ifneq ($(BUILDARCH),amd64)
	ETCD_IMAGE=$(ETCD_IMAGE)-$(ARCH)
endif

# TODO: Update this to use newer version of Kubernetes.
HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):v1.17.0
TEST_CONTAINER_FILES=$(shell find tests/ -type f ! -name '*.created' ! -name '*.pyc')

# Variables controlling the image
NODE_CONTAINER_CREATED=.calico_node.created-$(ARCH)
NODE_CONTAINER_BIN_DIR=./dist/bin/
NODE_CONTAINER_BINARY = $(NODE_CONTAINER_BIN_DIR)/calico-node-$(ARCH)
WINDOWS_BINARY = $(NODE_CONTAINER_BIN_DIR)/calico-node.exe
NODE_GIT_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
NODE_RELEASE_VERSION?=$(call git-release-tag-from-dev-tag)

# Variables for the Windows packaging.
# Name of the Windows release ZIP archive.
WINDOWS_ARCHIVE_ROOT := windows-packaging/CalicoWindows
WINDOWS_ARCHIVE_BINARY := $(WINDOWS_ARCHIVE_ROOT)/calico-node.exe
WINDOWS_ARCHIVE_TAG?=$(NODE_GIT_VERSION)
WINDOWS_ARCHIVE := dist/tigera-calico-windows-$(WINDOWS_ARCHIVE_TAG).zip
# Version of NSSM to download.
WINDOWS_NSSM_VERSION=2.24
# Explicit list of files that we copy in from the mod cache.  This is required because the copying rules we use are pattern-based
# and they only work with an explicit rule of the form "$(WINDOWS_MOD_CACHED_FILES): <file path from project root>" (otherwise,
# make has no way to know that the mod cache target produces the files we need).
WINDOWS_MOD_CACHED_FILES := \
    windows-packaging/config-bgp.ps1 \
    windows-packaging/config-bgp.psm1 \
    windows-packaging/conf.d/blocks.toml \
    windows-packaging/conf.d/peerings.toml \
    windows-packaging/templates/blocks.ps1.template \
    windows-packaging/templates/peerings.ps1.template \

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

MICROSOFT_SDN_VERSION := 0d7593e5c8d4c2347079a7a6dbd9eb034ae19a44
MICROSOFT_SDN_GITHUB_RAW_URL := https://raw.githubusercontent.com/microsoft/SDN/$(MICROSOFT_SDN_VERSION)

WINDOWS_UPGRADE_ROOT         ?= windows-upgrade
WINDOWS_UPGRADE_DIST          = dist/windows-upgrade

# The directory that holds temporary files used to build the windows upgrade zip
# archive.
WINDOWS_UPGRADE_DIST_STAGE    = $(WINDOWS_UPGRADE_DIST)/stage
WINDOWS_UPGRADE_INSTALL_FILE ?= $(WINDOWS_UPGRADE_DIST_STAGE)/install-calico-windows.ps1
WINDOWS_UPGRADE_INSTALL_ZIP  ?= $(WINDOWS_UPGRADE_DIST_STAGE)/calico-windows-$(WINDOWS_ARCHIVE_TAG).zip
WINDOWS_UPGRADE_SCRIPT       ?= $(WINDOWS_UPGRADE_DIST_STAGE)/calico-upgrade.ps1

# The directory used for the upgrade image docker build context.
WINDOWS_UPGRADE_BUILD        ?= $(WINDOWS_UPGRADE_ROOT)/build

# The final zip archive used in the upgrade image.
WINDOWS_UPGRADE_ARCHIVE      ?= $(WINDOWS_UPGRADE_BUILD)/calico-windows-upgrade.zip

# Variables used by the tests
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')
ST_TO_RUN?=tests/st/

# Can exclude the slower tests with "-a '!slow'"
ST_OPTIONS?=

# Variables for building the local binaries that go into the image
NODE_CONTAINER_FILES=$(shell find ./filesystem -type f)

# TODO(doublek): The various version variables in use here will need some cleanup.
# VERSION is used by cmd/calico-ipam and cmd/calico
# CNXVERSION is used by cmd/calico-node and pkg/lifecycle/startup
# CALICO_VERSION is used by pkg/lifecycle/startup
# All these are required for correct version reporting by the various binaries
# as well as embedding this information within the ClusterInformation resource.
LDFLAGS=-ldflags "\
	-X $(PACKAGE_NAME)/pkg/lifecycle/startup.CNXVERSION=$(NODE_GIT_VERSION) \
	-X $(PACKAGE_NAME)/pkg/lifecycle/startup.CNXRELEASEVERSION=$(NODE_RELEASE_VERSION) \
	-X $(PACKAGE_NAME)/pkg/lifecycle/startup.CALICOVERSION=$(CALICO_VERSION) \
	-X main.VERSION=$(NODE_GIT_VERSION) \
	-X $(PACKAGE_NAME)/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/buildinfo.BuildDate=$(DATE) \
	-X $(PACKAGE_NAME)/buildinfo.GitRevision=$(GIT_COMMIT)"

SRC_FILES=$(shell find ./pkg -name '*.go')

BINDIR?=bin

## Clean enough that a new release build will be clean
clean: clean-windows-upgrade
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
	rm -f $(WINDOWS_ARCHIVE_ROOT)/cni/*.exe
	rm -rf vendor crds.yaml
	rm -f $(WINDOWS_UPGRADE_INSTALL_FILE)
	rm -f $(WINDOWS_UPGRADE_BUILD)/*.zip
	rm -rf filesystem/included-source
	rm -rf dist
	rm -rf filesystem/etc/calico/confd/conf.d filesystem/etc/calico/confd/config filesystem/etc/calico/confd/templates
	rm -rf config/
	rm -rf vendor
	rm -rf bin
	rm Makefile.common*
	rm -rf bin
	# Delete images that we built in this repo
	docker rmi $(NODE_IMAGE):latest-$(ARCH) || true
	docker rmi $(TEST_CONTAINER_NAME) || true
	docker rmi $(addprefix $(WINDOWS_UPGRADE_IMAGE):latest-,$(WINDOWS_VERSIONS)) || true

clean-windows-upgrade:
	-rm -f "$(WINDOWS_UPGRADE_DIST_STAGE)"
	-rm -rf "$(WINDOWS_UPGRADE_BUILD)"

###############################################################################
# Updating pins
###############################################################################
API_REPO=github.com/tigera/api
LIBCALICO_REPO=github.com/tigera/libcalico-go-private
CONFD_REPO=github.com/tigera/confd-private
FELIX_REPO=github.com/tigera/felix-private
TYPHA_REPO=github.com/tigera/typha-private
CNI_REPO=github.com/tigera/cni-plugin-private

update-pins: update-api-pin replace-libcalico-pin update-confd-pin replace-felix-pin replace-typha-pin replace-cni-pin

###############################################################################
# Building the binary
###############################################################################
# build target is called from commit-pin-updates and it is essential that the
# MAKECMDGOALS remains as "commit-pin-updates" for various go flags to be set
# appropriately.
build: $(NODE_CONTAINER_BINARY)

remote-deps-copy-bpf: mod-download
	rm -rf bin/bpf
	mkdir -p bin/bpf
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
		$(GIT_CONFIG_SSH) \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-gpl bin/bpf; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-apache bin/bpf; \
		chmod -R +w bin/bpf'

.PHONY: remote-deps
remote-deps: remote-deps-copy-bpf
	# Recreate the directory so that we are sure to clean up any old files.
	rm -rf filesystem/etc/calico/confd
	mkdir -p filesystem/etc/calico/confd
	rm -rf config
	rm -rf bin/bpf
	mkdir -p bin/bpf
	rm -rf bin/third-party
	rm -rf filesystem/usr/lib/calico/bpf/
	mkdir -p filesystem/usr/lib/calico/bpf/
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
		$(GIT_CONFIG_SSH) \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/conf.d filesystem/etc/calico/confd/conf.d; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/config filesystem/etc/calico/confd/config; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/etc/calico/confd/templates filesystem/etc/calico/confd/templates; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/config config; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-gpl bin/bpf; \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-apache bin/bpf; \
		chmod -R +w bin/bpf; \
		chmod +x bin/bpf/bpf-gpl/list-* bin/bpf/bpf-gpl/calculate-*; \
		make -j 16 -C ./bin/bpf/bpf-apache/ all; \
		make -j 16 -C ./bin/bpf/bpf-gpl/ all; \
		cp bin/bpf/bpf-gpl/bin/* filesystem/usr/lib/calico/bpf/; \
		cp bin/bpf/bpf-apache/bin/* filesystem/usr/lib/calico/bpf/; \
		chmod -R +w filesystem/etc/calico/confd/ config/ filesystem/usr/lib/calico/bpf/'

$(LIBBPF_PATH)/libbpf.a: go.mod
	$(MAKE) mod-download
	mkdir -p bin/third-party
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
		$(GIT_CONFIG_SSH) \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-gpl/include/libbpf bin/third-party; \
		chmod -R +w bin/third-party; \
		make -j 16 -C $(LIBBPF_PATH) BUILD_STATIC_ONLY=1'

# We need CGO when compiling in Felix for BPF support.  However, the cross-compile doesn't support CGO yet.
# Currently CGO can be enbaled in ARM64 and AMD64 builds.
ifeq ($(ARCH), $(filter $(ARCH),amd64 arm64))
CGO_ENABLED=1
CGO_LDFLAGS="-L$(LIBBPF_PATH) -lbpf -lelf -lz"
CGO_CFLAGS="-I$(LIBBPF_DOCKER_PATH) -I$(BPF_GPL_DOCKER_PATH)"
else
CGO_ENABLED=0
CGO_LDFLAGS=""
CGO_CFLAGS=""
endif

DOCKER_GO_BUILD_CGO=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e CGO_LDFLAGS=$(CGO_LDFLAGS) -e CGO_CFLAGS=$(CGO_CFLAGS) $(CALICO_BUILD)
DOCKER_GO_BUILD_CGO_WINDOWS=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD)

$(NODE_CONTAINER_BINARY): remote-deps-copy-bpf $(LIBBPF_PATH)/libbpf.a $(LOCAL_BUILD_DEP) $(SRC_FILES) go.mod
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) go build -v -o $@ $(BUILD_FLAGS) $(LDFLAGS) ./cmd/calico-node/main.go'

$(WINDOWS_BINARY):
	$(DOCKER_GO_BUILD_CGO_WINDOWS) sh -c '$(GIT_CONFIG_SSH) \
		GOOS=windows CC=x86_64-w64-mingw32-gcc \
		go build --buildmode=exe -v -o $@ $(LDFLAGS) ./cmd/calico-node/main.go'

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
image: remote-deps $(NODE_IMAGE)
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*
ifeq ($(TEST_IMAGE_BUILD),true)
	# If testing image builds, clean sub-image afterwards to free disk space (for Semaphore CI)
	$(MAKE) clean-sub-image-$*
endif

## Remove images for all supported ARCHes
clean-image-all: $(addprefix clean-sub-image-,$(VALIDARCHES))
## Remove sub-image from docker and delete $(NODE_CONTAINER_CREATED) file
clean-sub-image-%:
	rm -f .calico_node.created-$*
	docker rmi $(NODE_IMAGE):latest-$* || true

$(NODE_IMAGE): $(NODE_CONTAINER_CREATED)
$(NODE_CONTAINER_CREATED): register ./Dockerfile.$(ARCH) $(NODE_CONTAINER_FILES) $(NODE_CONTAINER_BINARY) $(INCLUDED_SOURCE) remote-deps
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
## TARGET_PLATFORM fixes an issue where `FROM SCRATCH` in the Dockerfile share the same architecture as the host.
	docker build --pull -t $(NODE_IMAGE):latest-$(ARCH) $(TARGET_PLATFORM) . --build-arg BIRD_IMAGE=$(BIRD_IMAGE) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f ./Dockerfile.$(ARCH)
	touch $@

##########################################################################################
# TESTING
##########################################################################################

GINKGO_ARGS += -cover -timeout 20m --trace --v
GINKGO = ginkgo

#############################################
# Run unit level tests
#############################################
UT_PACKAGES_TO_SKIP?=pkg/lifecycle/startup,pkg/allocateip,pkg/status
.PHONY: ut
ut: CMD = go mod download && $(GINKGO) -r
ut:
ifdef LOCAL
	(CMD)
else
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(CMD) -skipPackage=$(UT_PACKAGES_TO_SKIP) $(GINKGO_ARGS)'
endif

# download BIRD source to include in image.
$(BIRD_SOURCE): go.mod
	mkdir -p filesystem/included-source/
	wget -O $@ https://github.com/projectcalico/bird/tarball/$(BIRD_VERSION)

# download any GPL felix code to include in the image.
$(FELIX_GPL_SOURCE): go.mod
	mkdir -p filesystem/included-source/
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c ' \
		tar cf $@ `go list -m -f "{{.Dir}}" github.com/projectcalico/felix`/bpf-gpl;'

###############################################################################
# FV Tests
###############################################################################
## Run the ginkgo FVs
fv: run-k8s-apiserver
	 $(DOCKER_RUN) -e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		ginkgo -cover -r -skipPackage vendor pkg/lifecycle/startup pkg/allocateip $(GINKGO_ARGS)'

## Create a local kind dual stack cluster.
KUBECONFIG?=kubeconfig.yaml
cluster-create: $(BINDIR)/kubectl $(BINDIR)/kind
	# First make sure any previous cluster is deleted
	make cluster-destroy
	
	# Create a kind cluster.
	$(BINDIR)/kind create cluster \
	        --config ./tests/kind-config.yaml \
	        --kubeconfig $(KUBECONFIG) \
	        --image kindest/node:$(K8S_VERSION)
	
	# Deploy resources needed in test env.
	$(MAKE) deploy-test-resources
	
	# Wait for controller manager to be running and healthy.
	while ! KUBECONFIG=$(KUBECONFIG) $(BINDIR)/kubectl get serviceaccount default; do echo "Waiting for default serviceaccount to be created..."; sleep 2; done

## Deploy resources on the kind cluster that are needed for tests
deploy-test-resources: $(BINDIR)/kubectl calico-node.tar
	KUBECONFIG=$(KUBECONFIG) GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) TSEE_TEST_LICENSE=$(TSEE_TEST_LICENSE) \
		   ./tests/k8st/deploy_resources_on_kind_cluster.sh

## Destroy local kind cluster
cluster-destroy: $(BINDIR)/kubectl $(BINDIR)/kind
	-$(BINDIR)/kubectl --kubeconfig=$(KUBECONFIG) drain kind-control-plane kind-worker kind-worker2 kind-worker3 --ignore-daemonsets --force
	-$(BINDIR)/kind delete cluster
	rm -f ./tests/k8st/infra/calico.yaml.tmp
	rm -f $(KUBECONFIG)

$(BINDIR)/kind:
	$(DOCKER_GO_BUILD) sh -c "GOBIN=/go/src/$(PACKAGE_NAME)/$(BINDIR) go install sigs.k8s.io/kind"

$(BINDIR)/kubectl:
	mkdir -p $(BINDIR)
	curl -L https://storage.googleapis.com/kubernetes-release/release/v1.22.0/bin/linux/$(ARCH)/kubectl -o $@
	chmod +x $(BINDIR)/kubectl

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
		${HYPERKUBE_IMAGE} kube-apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--logtostderr=true

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
	while ! docker exec st-apiserver kubectl \
		apply -f /manifests/config/crd/; \
		do echo "Trying to create CRDs"; \
		sleep 1; \
		done

# Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f st-apiserver

###############################################################################
# System tests
# - Support for running etcd (both securely and insecurely)
###############################################################################
# Pull calicoctl and CNI plugin binaries with versions as per XXX_VER
# variables.  These are used for the STs.
dist/calicoctl:
	mkdir -p dist
	-docker rm -f calicoctl
	docker pull $(CTL_CONTAINER_NAME)
	docker create --name calicoctl $(CTL_CONTAINER_NAME)
	docker cp calicoctl:calicoctl dist/calicoctl && \
	  test -e dist/calicoctl && \
	  touch dist/calicoctl
	-docker rm -f calicoctl

dist/calico dist/calico-ipam:
	mkdir -p dist
	-docker rm -f calico-cni
	docker pull $(CNX_REPOSITORY)/tigera/cni:$(CNI_VERSION)
	docker create --name calico-cni $(CNX_REPOSITORY)/tigera/cni:$(CNI_VERSION)
	docker cp calico-cni:/opt/cni/bin/install dist/calico && \
	  test -e dist/calico && \
	  touch dist/calico
	docker cp calico-cni:/opt/cni/bin/install dist/calico-ipam && \
	  test -e dist/calico-ipam && \
	  touch dist/calico-ipam
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
	docker run --rm $(NODE_IMAGE):latest-$(ARCH) /bin/sh -c "\
	  echo bird --version;	 /bin/bird --version; \
	"
	docker save --output $@ $(NODE_IMAGE):latest-$(ARCH)

calico-node.tar: cnx-node.tar
	cp cnx-node.tar calico-node.tar

.PHONY: st-checks
st-checks:
	# Check that we're running as root.
	test `id -u` -eq '0' || { echo "STs must be run as root to allow writes to /proc"; false; }

	# Insert an iptables rule to allow access from our test containers to etcd
	# running on the host.
	iptables-save | grep -q 'calico-st-allow-etcd' || iptables $(IPT_ALLOW_ETCD)

GCR_IO_PULL_SECRET?=${HOME}/.docker/config.json
TSEE_TEST_LICENSE?=${HOME}/secrets/new-test-customer-license.yaml

.PHONY: dual-tor-test
dual-tor-test: cnx-node.tar calico_test.created dual-tor-setup dual-tor-run-test dual-tor-cleanup

kubectl:
	curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.3/bin/linux/amd64/kubectl
	chmod +x ./kubectl

.PHONY: dual-tor-setup
DUAL_TOR_DIR=tests/k8st/dual-tor
dual-tor-setup: dual-tor-cleanup kubectl dist/calicoctl cnx-node.tar calico_test.created tests/k8st/reliable-nc/bin/reliable-nc
	docker build -t calico-test/busybox-with-reliable-nc tests/k8st/reliable-nc
	mkdir -p $(DUAL_TOR_DIR)/tmp
	cp -a cnx-node.tar $(DUAL_TOR_DIR)/tmp/
	docker build -t calico/dual-tor-node $(DUAL_TOR_DIR)
	rm -rf $(DUAL_TOR_DIR)/tmp
	GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) STEPS=setup \
	ROUTER_IMAGE=$(BIRD_IMAGE) CALICOCTL=`pwd`/dist/calicoctl $(DUAL_TOR_DIR)/dualtor.sh

DUAL_TOR_ST_TO_RUN=dual-tor-tests/test_dual_tor.py -s --nocapture --nologcapture
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
	     cd /code/tests/k8st && nosetests $(DUAL_TOR_ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

.PHONY: dual-tor-cleanup
dual-tor-cleanup:
	-STEPS=cleanup $(DUAL_TOR_DIR)/dualtor.sh

tests/k8st/reliable-nc/bin/reliable-nc: tests/k8st/reliable-nc/reliable-nc.go
	mkdir -p dist
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/tests/k8st/reliable-nc"'

## k8st: STs in a real Kubernetes cluster provisioned by KIND
##
## Note: if you're developing and want to see test output as it
## happens, instead of only later and if the test fails, add "-s
## --nocapture --nologcapture" to K8ST_TO_RUN.  For example:
##
## make k8s-test K8ST_TO_RUN="tests/test_dns_policy.py -s --nocapture --nologcapture"
##
## K8ST_RIG can be "dual_stack" or "vanilla".  "dual_stack" means set
## up for dual stack testing; it requires KIND changes that have not
## yet been merged to master, and runs kube-proxy in IPVS mode.
## "vanilla" means use vanilla upstream master KIND.
K8ST_RIG?=dual_stack
K8ST_TO_RUN?=-A $(K8ST_RIG)

.PHONY: k8s-test
k8s-test:
	$(MAKE) kind-k8st-setup
	$(MAKE) kind-k8st-run-test
	$(MAKE) kind-k8st-cleanup

.PHONY: kind-k8st-setup
kind-k8st-setup: calico-node.tar cluster-create

.PHONY: kind-k8st-run-test
kind-k8st-run-test: calico_test.created $(KUBECONFIG)
	docker run -t --rm \
	    -v $(CURDIR):/code \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v $(CURDIR)/$(KUBECONFIG):/root/.kube/config \
	    -v $(CURDIR)/$(BINDIR)/kubectl:/bin/kubectl \
	    -e ROUTER_IMAGE=$(BIRD_IMAGE) \
	    -e K8ST_RIG=$(K8ST_RIG) \
	    --privileged \
	    --net host \
	${TEST_CONTAINER_NAME} \
	    sh -c 'echo "container started.." && \
	     cd /code/tests/k8st && nosetests $(K8ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

.PHONY: kind-k8st-cleanup
kind-k8st-cleanup: cluster-destroy

# Needed for Semaphore CI (where disk space is a real issue during k8s-test)
.PHONY: remove-go-build-image
remove-go-build-image:
	@echo "Removing $(CALICO_BUILD) image to save space needed for testing ..."
	@-docker rmi $(CALICO_BUILD)

.PHONY: st
## Run the system tests
st: image remote-deps dist/calicoctl busybox.tar cnx-node.tar workload.tar run-etcd calico_test.created dist/calico dist/calico-ipam
	# Check versions of Calico binaries that ST execution will use.
	docker run --rm -v $(CURDIR)/dist:/go/bin:rw $(CALICO_BUILD) /bin/sh -c "\
	  echo; echo calicoctl version;	  /go/bin/calicoctl version; \
	  echo; echo calico -v;       /go/bin/calico -v; \
	  echo; echo calico-ipam -v;      /go/bin/calico-ipam -v; echo; \
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
		   -v $(TSEE_TEST_LICENSE):/license.yaml \
		   -e HOST_CHECKOUT_DIR=$(CURDIR) \
		   -e DEBUG_FAILURES=$(DEBUG_FAILURES) \
		   -e MY_IP=$(LOCAL_IP_ENV) \
		   -e NODE_CONTAINER_NAME=$(NODE_IMAGE):latest-$(ARCH) \
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

## Deploys images to registry
# Check-dirty before `cd` because `foss-checks` can lead to go.sum update.
# since `foss-checks` is defined in common Makefile, we do it just before `cd`.
cd: check-dirty cd-common

check-boring-ssl: $(NODE_CONTAINER_BIN_DIR)/calico-node-amd64
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD) \
		go tool nm $(NODE_CONTAINER_BIN_DIR)/calico-node-amd64 > $(NODE_CONTAINER_BIN_DIR)/tags.txt && grep '_Cfunc__goboringcrypto_' $(NODE_CONTAINER_BIN_DIR)/tags.txt 1> /dev/null
	-rm -f $(NODE_CONTAINER_BIN_DIR)/tags.txt

golangci-lint: $(GENERATED_FILES)
	$(DOCKER_GO_BUILD_CGO) golangci-lint run $(LINT_ARGS)

.PHONY: node-test-at
# Run docker-image acceptance tests
node-test-at: release-prereqs
	docker run -v $(PWD)/tests/at/calico_node_goss.yaml:/tmp/goss.yaml \
	  $(NODE_IMAGE):$(VERSION) /bin/sh -c ' \
	   apk --no-cache add wget ca-certificates && \
	   wget -q -O /tmp/goss https://github.com/aelsabbahy/goss/releases/download/v0.3.4/goss-linux-amd64 && \
	   chmod +rx /tmp/goss && \
	   /tmp/goss --gossfile /tmp/goss.yaml validate'

ensure-local-build-not-defined:
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif

ensure-calico-version-release-defined:
ifndef CALICO_VERSION_RELEASE
	$(error CALICO_VERSION_RELEASE is undefined - run using make release CALICO_VERSION_RELEASE=vX.Y.Z)
endif


###############################################################################
# Windows packaging
###############################################################################
# Pull the BGP configuration scripts and templates from the confd repo.
$(WINDOWS_MOD_CACHED_FILES): mod-download

$(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp%: windows-packaging/config-bgp%
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
        $(GIT_CONFIG_SSH) \
        cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/$< $@'; \
        chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/%: windows-packaging/conf.d/%
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
        $(GIT_CONFIG_SSH) \
        cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/$< $@'; \
        chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/confd/templates/%: windows-packaging/templates/%
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
        $(GIT_CONFIG_SSH) \
        cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/kelseyhightower/confd`/$< $@'; \
        chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1:
	wget -P $(WINDOWS_ARCHIVE_ROOT)/libs/hns/ $(MICROSOFT_SDN_GITHUB_RAW_URL)/Kubernetes/windows/hns.psm1

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt:
	wget -P $(WINDOWS_ARCHIVE_ROOT)/libs/hns/ $(MICROSOFT_SDN_GITHUB_RAW_URL)/License.txt

## Download NSSM.
windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip:
	wget -O windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip https://nssm.cc/release/nssm-$(WINDOWS_NSSM_VERSION).zip

build-windows-archive: $(WINDOWS_ARCHIVE_FILES) windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip check-dirty
	# To be as atomic as possible, we re-do work like unpacking NSSM here.
	-rm -f "$(WINDOWS_ARCHIVE)"
	-rm -rf $(WINDOWS_ARCHIVE_ROOT)/nssm-$(WINDOWS_NSSM_VERSION)
	mkdir -p dist
	cd windows-packaging && \
	cp -r CalicoWindows TigeraCalico && \
	sha256sum --check nssm.sha256sum && \
	cd TigeraCalico && \
	unzip  ../nssm-$(WINDOWS_NSSM_VERSION).zip \
	       -x 'nssm-$(WINDOWS_NSSM_VERSION)/src/*' && \
	cd .. && \
	zip -r "../$(WINDOWS_ARCHIVE)" TigeraCalico -x '*.git*'
	@echo
	@echo "Windows archive built at $(WINDOWS_ARCHIVE)"
	rm -rf windows-packaging/TigeraCalico

RELEASE_TAG_REGEX := ^v([0-9]{1,}\.){2}[0-9]{1,}$$
WINDOWS_GCS_BUCKET := gs://tigera-windows/dev/

# This target is just for Calico Enterprise. OS has a different release process.
# When merging, keep the 'release-windows-archive' target in private.
#
# This target builds the Windows installation zip file and uploads it to GCS.
push-windows-archive-gcs: build-windows-archive
	gcloud auth activate-service-account --key-file ~/secrets/gcp-registry-pusher-service-account.json
	gsutil cp dist/tigera-calico-windows-$(NODE_GIT_VERSION).zip $(WINDOWS_GCS_BUCKET)
	gcloud auth revoke registry-pusher@unique-caldron-775.iam.gserviceaccount.com

release-verify-version: var-require-all-VERSION
ifdef CONFIRM
	$(if $(filter $(VERSION),$(NODE_GIT_VERSION)),,\
		echo Current version $(CURRENT_RELEASE_VERSION) does not match given version $(VERSION) && exit 1)
endif

# Create and publish the windows archive for the release.
release-publish-windows-archive-gcs: var-require-one-of-CONFIRM-DRYRUN var-require-all-VERSION release-verify-version build-windows-archive
ifdef CONFIRM
	gcloud auth activate-service-account --key-file ~/secrets/gcp-registry-pusher-service-account.json
	gsutil cp dist/tigera-calico-windows-$(NODE_GIT_VERSION).zip gs://tigera-windows/
	gcloud auth revoke registry-pusher@unique-caldron-775.iam.gserviceaccount.com
else
	@echo [DRYRUN] gcloud auth activate-service-account --key-file ~/secrets/gcp-registry-pusher-service-account.json
	@echo [DRYRUN] gsutil cp dist/tigera-calico-windows-$(NODE_GIT_VERSION).zip gs://tigera-windows/
	@echo [DRYRUN] gcloud auth revoke registry-pusher@unique-caldron-775.iam.gserviceaccount.com
endif

$(WINDOWS_ARCHIVE_BINARY): $(WINDOWS_BINARY)
	cp $< $@

## Produces the Windows ZIP archive for the release.
## NOTE: this is needed to make the hash release, don't remove until that's changed.
release-windows-archive $(WINDOWS_ARCHIVE): var-require-all-VERSION
	$(MAKE) build-windows-archive WINDOWS_ARCHIVE_TAG=$(VERSION)

# Ensure the upgrade image docker build folder exists.
$(WINDOWS_UPGRADE_BUILD):
	-mkdir -p $(WINDOWS_UPGRADE_BUILD)

# Ensure the directory for temporary files used to build the windows upgrade zip
# archive exists.
$(WINDOWS_UPGRADE_DIST_STAGE):
	-mkdir -p $(WINDOWS_UPGRADE_DIST_STAGE)

# Copy the upgrade script to the temporary directory where we build the windows
# upgrade zip file.
$(WINDOWS_UPGRADE_SCRIPT): $(WINDOWS_UPGRADE_DIST_STAGE)
	cp $(WINDOWS_UPGRADE_ROOT)/calico-upgrade.ps1 $@

# Copy the install zip archive to the temporary directory where we build the windows
# upgrade zip file.
$(WINDOWS_UPGRADE_INSTALL_ZIP): build-windows-archive $(WINDOWS_UPGRADE_DIST_STAGE)
	cp $(WINDOWS_ARCHIVE) $@

# Get the install script into the temporary directory where we build the windows
# upgrade zip file. The version of the install script depends on whether there
# is a released version for the branch; otherwise the master version of the
# script is used.
$(WINDOWS_UPGRADE_INSTALL_FILE): $(WINDOWS_UPGRADE_DIST_STAGE)
	# Truncated git version in the vX.Y version string our docs site uses.
	$(eval ver := $(shell echo $(WINDOWS_ARCHIVE_TAG) | sed -ne 's/\(v[0-9]\+\.[0-9]\+\).*/\1/p' ))
	# The vX.Y.Z version string.
	$(eval fullver := $(shell echo $(WINDOWS_ARCHIVE_TAG) | sed -ne 's/\(v[0-9]\+\.[0-9]\+\.[0-9]\+\).*/\1/p' ))
	@echo vX.Y version is $(ver)
	@echo vX.Y.Z version is $(fullver)
	@if git show-ref --tags $(fullver) ; then \
		echo "Tag $(fullver) exists, using released version of installation script" ; \
		curl --fail https://docs.tigera.io/$(ver)/scripts/install-calico-windows.ps1 -o $(WINDOWS_UPGRADE_INSTALL_FILE) ; \
	else \
		echo "Tag $(fullver) doesn't exist yet, using master version of installation script" ; \
		curl --fail https://docs.tigera.io/master/scripts/install-calico-windows.ps1 -o $(WINDOWS_UPGRADE_INSTALL_FILE) ; \
	fi

# Produces the Windows upgrade ZIP archive for the release.
release-windows-upgrade-archive: release-prereqs
	$(MAKE) build-windows-upgrade-archive WINDOWS_ARCHIVE_TAG=$(VERSION)

# Build the Windows upgrade zip archive.
build-windows-upgrade-archive: clean-windows-upgrade $(WINDOWS_UPGRADE_INSTALL_ZIP) $(WINDOWS_UPGRADE_INSTALL_FILE) $(WINDOWS_UPGRADE_SCRIPT) $(WINDOWS_UPGRADE_BUILD)
	rm $(WINDOWS_UPGRADE_ARCHIVE) || true
	cd $(WINDOWS_UPGRADE_DIST_STAGE) && zip -r "$(CURDIR)/$(WINDOWS_UPGRADE_ARCHIVE)" *.zip *.ps1

# Sets up the docker builder used to create Windows image tarballs.
setup-windows-builder:
	-docker buildx rm calico-windows-upgrade-builder
	docker buildx create --name=calico-windows-upgrade-builder --use --platform windows/amd64

# Builds all the Windows image tarballs for each version in WINDOWS_VERSIONS
image-tar-windows-all: setup-windows-builder $(addprefix sub-image-tar-windows-,$(WINDOWS_VERSIONS))

CRANE_BINDMOUNT_CMD := \
	docker run --rm \
		--net=host \
		--init \
		--entrypoint /bin/sh \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-v $(DOCKER_CONFIG):/root/.docker/config.json \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) -c $(double_quote)crane

DOCKER_MANIFEST_CMD := docker manifest

ifdef CONFIRM
CRANE_BINDMOUNT = $(CRANE_BINDMOUNT_CMD)
DOCKER_MANIFEST = $(DOCKER_MANIFEST_CMD)
else
CRANE_BINDMOUNT = echo [DRY RUN] $(CRANE_BINDMOUNT_CMD)
DOCKER_MANIFEST = echo [DRY RUN] $(DOCKER_MANIFEST_CMD)
endif

# Uses the docker builder to create a Windows image tarball for a single Windows
# version.
sub-image-tar-windows-%:
	-mkdir -p $(WINDOWS_UPGRADE_DIST)
	cd $(WINDOWS_UPGRADE_ROOT) && \
		docker buildx build \
			--platform windows/amd64 \
			--output=type=docker,dest=$(CURDIR)/$(WINDOWS_UPGRADE_DIST)/image-$(GIT_VERSION)-$*.tar \
			--pull \
			--no-cache \
			--build-arg=WINDOWS_VERSION=$* .

# The calico-windows-upgrade cd is different because we do not build docker images directly.
# Since the build machine is linux, we output the images to a tarball. (We can
# produce images but there will be no output because docker images
# built for Windows cannot be loaded on linux.)
#
# The resulting image tarball is then pushed to registries during cd/release.
# The image tarballs are located in dist/windows-upgrade and have files names
# with the format 'image-v3.21.0-2-abcdef-20H2.tar'.
#
# In addition to pushing the individual images, we also create the manifest
# directly using 'docker manifest'. This is possible because Semaphore is using
# a recent enough docker CLI version (20.10.0)
#
# - Create the manifest with 'docker manifest create' using the list of all images.
# - For each windows version and image tag combination, 'docker manifest annotate' its image with "os.image: <windows_version>".
#   <windows_version> is the version string that looks like, e.g. 10.0.19041.1288.
#   Setting os.image in the manifest is required for Windows hosts to load the
#   correct image in manifest.
# - Finally we push the manifest, "purging" the local manifest.
cd-windows-upgrade:
	tags="$(GIT_VERSION) $(BRANCH_NAME)"; \
	for registry in $(DEV_REGISTRIES); do \
		for tag in $${tags}; do \
			echo Pushing Windows images to $${registry}; \
			all_images=""; \
			manifest_image="$${registry}/$(WINDOWS_UPGRADE_IMAGE):$${tag}"; \
			for win_ver in $(WINDOWS_VERSIONS); do \
				image_tar="$(WINDOWS_UPGRADE_DIST)/image-$(GIT_VERSION)-$${win_ver}.tar"; \
				image="$${registry}/$(WINDOWS_UPGRADE_IMAGE):$${tag}-windows-$${win_ver}"; \
				echo Pushing image $${image} ...; \
				$(CRANE_BINDMOUNT) push $${image_tar} $${image}$(double_quote) & \
				all_images="$${all_images} $${image}"; \
			done; \
			wait; \
			$(DOCKER_MANIFEST) create --amend $${manifest_image} $${all_images}; \
			for win_ver in $(WINDOWS_VERSIONS); do \
				version=$$(docker manifest inspect mcr.microsoft.com/windows/nanoserver:$${win_ver} | grep "os.version" | head -n 1 | awk -F\" '{print $$4}'); \
				image="$${registry}/$(WINDOWS_UPGRADE_IMAGE):$${tag}-windows-$${win_ver}"; \
				$(DOCKER_MANIFEST) annotate --os windows --arch amd64 --os-version $${version} $${manifest_image} $${image}; \
			done; \
			$(DOCKER_MANIFEST) push --purge $${manifest_image}; \
		done; \
	done ;

###############################################################################
# Utilities
###############################################################################
$(info "Build dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "BIRD_VERSION" $(BIRD_VERSION)))

$(info "Test dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "CNI_VERSION" $(CNI_VERSION)))

$(info "Calico git version")
$(info $(shell printf "%-21s = %-10s\n" "GIT_VERSION" $(GIT_VERSION)))
