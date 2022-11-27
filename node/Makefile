include ../metadata.mk

PACKAGE_NAME = github.com/projectcalico/calico/node

# Name of the images.
# e.g., <registry>/<name>:<tag>
NODE_IMAGE            ?=cnx-node
# Name of the Enterprise windows image
# The windows image names are:
# - OS: calico/windows
# - EE: tigera/calico-windows
WINDOWS_IMAGE         ?=calico-windows
# Name of the Enterprise windows upgrade image
# The windows upgrade image names are:
# - OS: calico/windows-upgrade
# - EE: tigera/calico-windows-upgrade
WINDOWS_UPGRADE_IMAGE ?=calico-windows-upgrade

# If this is a windows release build the windows upgrade image.
# ARCHES will contain the values in WINDOWS_VERSIONS, prefixed with "windows-".
WINDOWS_VERSIONS      ?=1809 2004 20H2 ltsc2022
ifdef WINDOWS_RELEASE
BUILD_IMAGES          ?=$(WINDOWS_IMAGE) $(WINDOWS_UPGRADE_IMAGE)
ARCHES                ?= $(patsubst %,windows-%,$(WINDOWS_VERSIONS))
else
BUILD_IMAGES          ?=$(NODE_IMAGE)
endif

RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# Paths within the build container for BPF source.
LIBBPF_CONTAINER_PATH=/go/src/github.com/projectcalico/calico/felix/bpf-gpl/include/libbpf/src/
BPFGPL_CONTAINER_PATH=/go/src/github.com/projectcalico/calico/felix/bpf-gpl/

# Paths within the repository for BPF source.
LIBBPF_A=../felix/bpf-gpl/include/libbpf/src/$(ARCH)/libbpf.a

# These arches not yet building in node-private
EXCLUDEARCH?=s390x arm64 ppc64le

# This gets embedded into node as the Calico version, the Enterprise release
# is based off of. This should be updated everytime a new opensource Calico
# release is merged into node-private.
CALICO_VERSION=v3.22.0

# Add in local static-checks
LOCAL_CHECKS=check-boring-ssl

# Complete list of files from other directories that we need to build calico/node.
REMOTE_DEPS = $(LIBBPF_A) \
	      filesystem/usr/lib/calico/bpf \
	      filesystem/etc/calico/confd/conf.d \
	      filesystem/etc/calico/confd/templates

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

# Set the platform correctly for building docker images.
ifeq ($(ARCH),arm64)
# Required for eBPF support in ARM64.
# We need to force ARM64 build image to be used in a crosscompilation run.
CALICO_BUILD:=$(CALICO_BUILD)-$(ARCH)
endif

###############################################################################

# Versions and location of dependencies used in the build.
BIRD_IMAGE ?= calico/bird:$(BIRD_VERSION)-$(ARCH)
BIRD_SOURCE=filesystem/included-source/bird-$(BIRD_VERSION).tar.gz
FELIX_GPL_SOURCE=filesystem/included-source/felix-ebpf-gpl.tar.gz
INCLUDED_SOURCE=$(BIRD_SOURCE) $(FELIX_GPL_SOURCE)

# Versions and locations of dependencies used in tests.
TEST_CONTAINER_NAME_VER?=latest
TEST_CONTAINER_NAME?=calico/test:$(TEST_CONTAINER_NAME_VER)-$(ARCH)

TEST_CONTAINER_FILES=$(shell find tests/ -type f ! -name '*.created' ! -name '*.pyc')

# Variables controlling the image
NODE_CONTAINER_CREATED=.calico_node.created-$(ARCH)
NODE_CONTAINER_BIN_DIR=./dist/bin/
NODE_CONTAINER_BINARY = $(NODE_CONTAINER_BIN_DIR)/calico-node-$(ARCH)
WINDOWS_BINARY = $(NODE_CONTAINER_BIN_DIR)/calico-node.exe
TOOLS_MOUNTNS_BINARY = $(NODE_CONTAINER_BIN_DIR)/mountns-$(ARCH)
NODE_GIT_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
NODE_RELEASE_VERSION?=$(call git-release-tag-from-dev-tag)

WINDOWS_INSTALL_SCRIPT := dist/install-calico-windows.ps1

# Variables for the Windows packaging.
# Name of the Windows release ZIP archive.
WINDOWS_PACKAGING_ROOT := windows-packaging
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
# because we need to force them to be built/copied into place. We also have
# tests in windows-packaging that we don't want to include.
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

# The directory for temporary files used to build the windows upgrade zip archive.
WINDOWS_UPGRADE_DIST_STAGE    = $(WINDOWS_UPGRADE_DIST)/stage

# Windows upgrade archive components.
WINDOWS_UPGRADE_INSTALL_FILE ?= $(WINDOWS_UPGRADE_DIST_STAGE)/install-calico-windows.ps1
WINDOWS_UPGRADE_INSTALL_ZIP  ?= $(WINDOWS_UPGRADE_DIST_STAGE)/calico-windows-$(WINDOWS_ARCHIVE_TAG).zip
WINDOWS_UPGRADE_SCRIPT       ?= $(WINDOWS_UPGRADE_DIST_STAGE)/calico-upgrade.ps1

# The directory for the upgrade image docker build context.
WINDOWS_UPGRADE_BUILD        ?= $(WINDOWS_UPGRADE_ROOT)/build

# The final zip archive used in the upgrade image.
WINDOWS_UPGRADE_ARCHIVE      ?= $(WINDOWS_UPGRADE_BUILD)/calico-windows-upgrade.zip

# The directory for windows image tarball
WINDOWS_DIST        = dist/windows

# The directory for temporary files used to build the windows image
WINDOWS_DIST_STAGE  = $(WINDOWS_DIST)/stage

# Variables used by the tests
ST_TO_RUN?=tests/st/

# Can exclude the slower tests with "-a '!slow'"
ST_OPTIONS?=

# Filesystem of the node container that is checked in to this repository.
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

# Source golang files on which compiling the calico-node binary depends.
SRC_FILES=$(shell find ./pkg -name '*.go') \
	  $(shell find ../felix -name '*.go') \
	  $(shell find ../felix -name '*.[ch]') \
	  $(shell find ../libcalico-go -name '*.go') \
	  $(shell find ../confd -name '*.go')

BINDIR?=bin

## Clean enough that a new release build will be clean
clean: clean-windows
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf .go-pkg-cache
	rm -rf certs *.tar $(NODE_CONTAINER_BIN_DIR)
	rm -rf $(REMOTE_DEPS)
	rm -rf filesystem/included-source
	rm -rf dist
	rm -rf bin
	# We build felix as part of the node build, so clean it as part of the clean.
	make -C ../felix clean
	# Delete images that we built in this repo
	docker rmi $(NODE_IMAGE):latest-$(ARCH) || true
	docker rmi $(TEST_CONTAINER_NAME) || true
	docker rmi $(addprefix $(WINDOWS_UPGRADE_IMAGE):latest-,$(WINDOWS_VERSIONS)) || true

clean-windows:
	-rm -f $(WINDOWS_ARCHIVE) $(WINDOWS_ARCHIVE_BINARY) $(WINDOWS_BINARY)
	-rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1
	-rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt
	-rm -f $(WINDOWS_ARCHIVE_ROOT)/cni/*.exe
	-rm -f $(WINDOWS_INSTALL_SCRIPT)
	-rm -rf "$(WINDOWS_DIST)"
	-rm -rf "$(WINDOWS_UPGRADE_DIST)"
	-rm -rf "$(WINDOWS_UPGRADE_BUILD)"

###############################################################################
# Building the binary
###############################################################################
build: $(NODE_CONTAINER_BINARY) $(TOOLS_MOUNTNS_BINARY)

# Pull in config from confd.
filesystem/etc/calico/confd/conf.d: $(shell find ../confd/etc/calico/confd/conf.d -type f)
	rm -rf $@ && cp -r ../confd/etc/calico/confd/conf.d $@
	chmod +w $@

filesystem/etc/calico/confd/templates: $(shell find ../confd/etc/calico/confd/templates -type f)
	rm -rf $@ && cp -r ../confd/etc/calico/confd/templates $@
	chmod +w $@

$(LIBBPF_A): $(shell find ../felix/bpf-gpl/include/libbpf -type f -name '*.[ch]')
	make -C ../felix libbpf ARCH=$(ARCH)

filesystem/usr/lib/calico/bpf: $(shell find ../felix/bpf-gpl -type f) $(shell find ../felix/bpf-apache -type f)
	rm -rf filesystem/usr/lib/calico/bpf/ && mkdir -p filesystem/usr/lib/calico/bpf/
	make -C ../felix build-bpf ARCH=$(ARCH)
	cp -r ../felix/bpf-gpl/bin/* $@
	cp -r ../felix/bpf-apache/bin/* $@

# We need CGO when compiling in Felix for BPF support.
# Currently CGO can be enabled in ARM64 and AMD64 builds.
ifeq ($(ARCH), $(filter $(ARCH),amd64 arm64))
CGO_ENABLED=1
CGO_LDFLAGS="-L$(LIBBPF_CONTAINER_PATH)/$(ARCH) -lbpf -lelf -lz"
CGO_CFLAGS="-I$(LIBBPF_CONTAINER_PATH) -I$(BPFGPL_CONTAINER_PATH)"
else
CGO_ENABLED=0
CGO_LDFLAGS=""
CGO_CFLAGS=""
endif

DOCKER_GO_BUILD_CGO=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e CGO_LDFLAGS=$(CGO_LDFLAGS) -e CGO_CFLAGS=$(CGO_CFLAGS) $(CALICO_BUILD)
DOCKER_GO_BUILD_CGO_WINDOWS=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD)

$(NODE_CONTAINER_BINARY): filesystem/usr/lib/calico/bpf $(LIBBPF_A) $(SRC_FILES) ../go.mod
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) go build -v -o $@ $(BUILD_FLAGS) $(LDFLAGS) ./cmd/calico-node/main.go'

$(WINDOWS_BINARY):
	$(DOCKER_GO_BUILD_CGO_WINDOWS) sh -c '$(GIT_CONFIG_SSH) \
		GOOS=windows CC=x86_64-w64-mingw32-gcc \
		go build --buildmode=exe -v -o $@ $(LDFLAGS) ./cmd/calico-node/main.go'

$(WINDOWS_ARCHIVE_ROOT)/cni/calico.exe:
	$(DOCKER_RUN) \
		-e GOOS=windows \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(LDFLAGS) ./cmd/calico'

$(WINDOWS_ARCHIVE_ROOT)/cni/calico-ipam.exe:
	$(DOCKER_RUN) \
		-e GOOS=windows \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $@ $(LDFLAGS) ./cmd/calico-ipam'

$(TOOLS_MOUNTNS_BINARY):
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) go build -v -o $@ $(BUILD_FLAGS) $(LDFLAGS) ./cmd/mountns'

###############################################################################
# Building the image
###############################################################################
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image $(NODE_IMAGE): register $(NODE_CONTAINER_CREATED)
$(NODE_CONTAINER_CREATED): $(REMOTE_DEPS) ./Dockerfile.$(ARCH) $(NODE_CONTAINER_BINARY) $(INCLUDED_SOURCE) $(NODE_CONTAINER_FILES) $(TOOLS_MOUNTNS_BINARY)
	$(DOCKER_BUILD) --build-arg BIRD_IMAGE=$(BIRD_IMAGE) -t $(NODE_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

##########################################################################################
# TESTING
##########################################################################################

GINKGO_ARGS += -cover -timeout 20m --trace --v
GINKGO = ginkgo

#############################################
# Run unit level tests
#############################################
# Skip packages containing FV tests.
UT_PACKAGES_TO_SKIP?=pkg/lifecycle/startup,pkg/allocateip,pkg/status
.PHONY: ut
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) ginkgo -r -skipPackage=$(UT_PACKAGES_TO_SKIP) $(GINKGO_ARGS)'

# download BIRD source to include in image.
$(BIRD_SOURCE): .bird-source.created
.bird-source.created:
	rm -rf filesystem/included-source/bird*
	mkdir -p filesystem/included-source/
	wget -O $(BIRD_SOURCE) https://github.com/projectcalico/bird/tarball/$(BIRD_VERSION)
	touch $@

# include GPL felix code in the image.
$(FELIX_GPL_SOURCE):
.felix-gpl-source.created: $(shell find ../felix/bpf-gpl -type f)
	rm -rf filesystem/included-source/felix*
	mkdir -p filesystem/included-source/
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'tar cf $(FELIX_GPL_SOURCE) ../felix/bpf-gpl;'
	touch $@

###############################################################################
# FV Tests
###############################################################################
K8ST_IMAGE_TARS=calico-node.tar calico-apiserver.tar cnx-node.tar calico-cni.tar pod2daemon.tar calicoctl.tar kube-controllers.tar egress-gateway.tar

ifeq ($(SEMAPHORE_GIT_REF_TYPE), pull-request)
# Determine the tests to run using the test spider tool, which emits a list of impacted packages.
WHAT=$(shell $(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go run ../hack/test/spider -commit-range=${SEMAPHORE_GIT_COMMIT_RANGE} -filter-dir node/')
else
# By default, run all tests.
WHAT=$(shell find . -name "*_test.go" | xargs dirname | sort -u)
endif

## Run the ginkgo tests.
ut fv: run-k8s-apiserver
	$(DOCKER_RUN) \
	-v $(CERTS_PATH):/home/user/certs \
	-e KUBECONFIG=/go/src/github.com/projectcalico/calico/hack/test/certs/kubeconfig \
	-e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 \
	$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) ./run-uts $(WHAT)'

###############################################################################
# System tests
###############################################################################
dist/calicoctl:
	mkdir -p dist
	make -C ../calicoctl build
	cp ../calicoctl/bin/calicoctl-linux-$(ARCH) $@

dist/calico dist/calico-ipam:
	mkdir -p dist
	make -C ../cni-plugin build
	cp ../cni-plugin/bin/$(ARCH)/calico dist/calico
	cp ../cni-plugin/bin/$(ARCH)/calico-ipam dist/calico-ipam

# Create images for containers used in the tests
busybox.tar:
	docker pull $(ARCH)/busybox:latest
	docker save --output busybox.tar $(ARCH)/busybox:latest

workload.tar:
	cd workload && $(DOCKER_BUILD) -t workload -f Dockerfile.$(ARCH) . --load
	docker save --output workload.tar workload

IPT_ALLOW_ETCD:=-A INPUT -i docker0 -p tcp --dport 2379 -m comment --comment "calico-st-allow-etcd" -j ACCEPT

# Create the calico/test image
test_image: .calico_test.created
.calico_test.created: $(TEST_CONTAINER_FILES)
	cd calico_test && $(DOCKER_BUILD) -f Dockerfile.$(ARCH).calico_test -t $(TEST_CONTAINER_NAME) . --load
	touch $@

cnx-node.tar: $(NODE_CONTAINER_CREATED)
	docker save --output $@ $(NODE_IMAGE):latest-$(ARCH)

calico-node.tar: cnx-node.tar
	cp cnx-node.tar calico-node.tar

calico-apiserver.tar: ../go.mod $(shell find ../apiserver -name '*.go') $(shell find ../libcalico-go -name '*.go')
	make -C ../apiserver image
	docker save --output $@ tigera/cnx-apiserver:latest-$(ARCH)

calico-cni.tar: ../go.mod $(shell find ../cni-plugin -name '*.go') $(shell find ../libcalico-go -name '*.go')
	make -C ../cni-plugin image
	docker save --output $@ tigera/cni:latest-$(ARCH)

pod2daemon.tar: ../go.mod $(shell find ../pod2daemon -name '*.go')
	make -C ../pod2daemon image
	docker save --output $@ tigera/pod2daemon-flexvol:latest-$(ARCH)

calicoctl.tar: ../go.mod $(shell find ../calicoctl -name '*.go') $(shell find ../libcalico-go -name '*.go')
	make -C ../calicoctl image
	docker save --output $@ tigera/calicoctl:latest-$(ARCH)

kube-controllers.tar: ../go.mod $(shell find ../kube-controllers -name '*.go') $(shell find ../libcalico-go -name '*.go')
	make -C ../kube-controllers image
	docker save --output $@ tigera/kube-controllers:latest-$(ARCH)

egress-gateway.tar: ../go.mod $(shell find ../egress-gateway -name '*.go') ../egress-gateway/* $(shell find ../libcalico-go -name '*.go')
	make -C ../egress-gateway image
	docker save --output $@ tigera/egress-gateway:latest-$(ARCH)

load-container-images: $(K8ST_IMAGE_TARS) $(KUBECTL)
	# Load the latest tar files onto the currently running kind cluster.
	KUBECONFIG=$(KIND_KUBECONFIG) ./tests/k8st/load_images_on_kind_cluster.sh
	# Restart the Calico containers so they launch with the newly loaded code.
	# TODO: We should be able to do this without restarting everything in kube-system.
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL) delete pods -n kube-system --all
	# calicoctl is deployed as a pod on the cluster and needs to be recreated.
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL) apply -f tests/k8st/infra/calicoctl.yaml

.PHONY: st-checks
st-checks:
	# Check that we're running as root.
	test `id -u` -eq '0' || { echo "STs must be run as root to allow writes to /proc"; false; }

	# Insert an iptables rule to allow access from our test containers to etcd
	# running on the host.
	iptables-save | grep -q 'calico-st-allow-etcd' || iptables $(IPT_ALLOW_ETCD)

GCR_IO_PULL_SECRET?=${HOME}/secrets/docker_cfg.json
TSEE_TEST_LICENSE?=${HOME}/secrets/new-test-customer-license.yaml

.PHONY: dual-tor-test
dual-tor-test: cnx-node.tar .calico_test.created dual-tor-setup dual-tor-run-test dual-tor-cleanup

kubectl:
	curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.3/bin/linux/amd64/kubectl
	chmod +x ./kubectl

# TODO: dual-tor uses a custom kind build, and so doesn't leverage the common
# kind logic from lib.Makefile. 
.PHONY: dual-tor-setup
DUAL_TOR_DIR=tests/k8st/dual-tor
dual-tor-setup: dual-tor-cleanup kubectl dist/calicoctl $(K8ST_IMAGE_TARS) .calico_test.created tests/k8st/reliable-nc/bin/reliable-nc
	docker build -t calico-test/busybox-with-reliable-nc tests/k8st/reliable-nc
	mkdir -p $(DUAL_TOR_DIR)/tmp
	cp -a $(K8ST_IMAGE_TARS) $(DUAL_TOR_DIR)/tmp/
	docker build --build-arg K8S_VERSION=$(K8S_VERSION) -t calico/dual-tor-node $(DUAL_TOR_DIR)
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

.PHONY: external-network-setup
EXTERNAL_NETWORK_DIR=tests/k8st/external-network
# external-network-setup: external-network-cleanup kubectl dist/calicoctl $(K8ST_IMAGE_TARS) .calico_test.created tests/k8st/reliable-nc/bin/reliable-nc
external-network-setup: external-network-cleanup kubectl dist/calicoctl
	GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) STEPS=setup \
	ROUTER_IMAGE=$(BIRD_IMAGE) CALICOCTL=`pwd`/dist/calicoctl K8S_VERSION=$(K8S_VERSION) $(EXTERNAL_NETWORK_DIR)/external-networks.sh

.PHONY: external-network-cleanup
external-network-cleanup:
	-STEPS=cleanup $(EXTERNAL_NETWORK_DIR)/external-networks.sh

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
## By default, run tests tagged with "vanilla". This excludes dual tor tests from running
## on a test environment which doesn't support dual tor, and excludes egress IP tests which
## reconfigure the cluster configuration.
K8ST_TO_RUN ?= -A vanilla
EGRESS_KIND_CONFIG=./tests/k8st/egress-ip-tests/kind.config

.PHONY: egress-ip-test
egress-ip-test:
	KIND_CONFIG=$(EGRESS_KIND_CONFIG) $(MAKE) k8s-test

.PHONY: k8s-test
k8s-test:
	$(MAKE) kind-k8st-setup
	$(MAKE) kind-k8st-run-test
	$(MAKE) kind-k8st-cleanup

.PHONY: kind-k8st-setup
TEST_LICENSE=$(SECRETS_PATH)/license.yaml
kind-k8st-setup: $(K8ST_IMAGE_TARS) kind-cluster-create $(TEST_LICENSE)
	TSEE_TEST_LICENSE=$(TEST_LICENSE) GCR_IO_PULL_SECRET=$(GCR_IO_PULL_SECRET) KUBECONFIG=$(KIND_KUBECONFIG) ARCH=$(ARCH) ./tests/k8st/deploy_resources_on_kind_cluster.sh

.PHONY: kind-k8st-run-test
kind-k8st-run-test: .calico_test.created $(KIND_KUBECONFIG)
	docker run -t --rm \
	    -v $(CURDIR):/code \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v $(KIND_KUBECONFIG):/root/.kube/config \
	    -v $(KUBECTL):/bin/kubectl \
	    -e ROUTER_IMAGE=$(BIRD_IMAGE) \
	    --privileged \
	    --net host \
	${TEST_CONTAINER_NAME} \
	    sh -c 'echo "container started.." && \
	     cd /code/tests/k8st && nosetests $(K8ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

.PHONY: kind-k8st-cleanup
kind-k8st-cleanup: kind-cluster-destroy

# Needed for Semaphore CI (where disk space is a real issue during k8s-test)
.PHONY: remove-go-build-image
remove-go-build-image:
	@echo "Removing $(CALICO_BUILD) image to save space needed for testing ..."
	@-docker rmi $(CALICO_BUILD)

.PHONY: st
## Run the system tests
st: $(REMOTE_DEPS) image dist/calicoctl busybox.tar calico-node.tar workload.tar run-etcd .calico_test.created dist/calico dist/calico-ipam
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
ci: static-checks ut image-all build-windows-upgrade-archive image-tar-windows-all st

## Deploys images to registry
cd: cd-common cd-windows-all

check-boring-ssl: $(NODE_CONTAINER_BIN_DIR)/calico-node-amd64
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD) \
		go tool nm $(NODE_CONTAINER_BIN_DIR)/calico-node-amd64 > $(NODE_CONTAINER_BIN_DIR)/tags.txt && grep '_Cfunc__goboringcrypto_' $(NODE_CONTAINER_BIN_DIR)/tags.txt 1> /dev/null
	-rm -f $(NODE_CONTAINER_BIN_DIR)/tags.txt

# Override the lib.Makefile golangci-lint target to use CGO, since some packages we pull in
# require it. Without this, we get undefined value errors.
golangci-lint: $(GENERATED_FILES)
	$(DOCKER_GO_BUILD_CGO) golangci-lint run $(LINT_ARGS)

.PHONY: node-test-at
# Run docker-image acceptance tests
node-test-at: release-prereqs
	docker run -v $(CURDIR)/tests/at/calico_node_goss.yaml:/tmp/goss.yaml \
	  $(NODE_IMAGE):$(VERSION) /bin/sh -c ' \
	   apk --no-cache add wget ca-certificates && \
	   wget -q -O /tmp/goss https://github.com/aelsabbahy/goss/releases/download/v0.3.4/goss-linux-amd64 && \
	   chmod +rx /tmp/goss && \
	   /tmp/goss --gossfile /tmp/goss.yaml validate'

ensure-calico-version-release-defined:
ifndef CALICO_VERSION_RELEASE
	$(error CALICO_VERSION_RELEASE is undefined - run using make release CALICO_VERSION_RELEASE=vX.Y.Z)
endif

###############################################################################
# Windows packaging
###############################################################################
# Pull the BGP configuration scripts and templates from the confd repo.
$(WINDOWS_MOD_CACHED_FILES):

$(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp%: windows-packaging/config-bgp%
	cp -r ../confd/$< $@
	chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/%: windows-packaging/conf.d/%
	cp -r ../confd/$< $@
	chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/confd/templates/%: windows-packaging/templates/%
	cp -r ../confd/$< $@
	chmod +w $@

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1:
	wget -P $(WINDOWS_ARCHIVE_ROOT)/libs/hns/ $(MICROSOFT_SDN_GITHUB_RAW_URL)/Kubernetes/windows/hns.psm1

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt:
	wget -P $(WINDOWS_ARCHIVE_ROOT)/libs/hns/ $(MICROSOFT_SDN_GITHUB_RAW_URL)/License.txt

## Download NSSM.
windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip:
	wget -O windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip https://nssm.cc/release/nssm-$(WINDOWS_NSSM_VERSION).zip

build-windows-archive: $(WINDOWS_ARCHIVE_FILES) windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip
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

# Build the docs site and copy over the install-calico-windows.ps1 script.
$(WINDOWS_INSTALL_SCRIPT):
	-mkdir -p dist
	make -C ../calico clean _site
	cp $(CURDIR)/../calico/_site/scripts/install-calico-windows.ps1 $@

# Copy the install-calico-windows.ps1 script to the temporary directory where we
# build the windows upgrade zip file.
$(WINDOWS_UPGRADE_INSTALL_FILE): $(WINDOWS_UPGRADE_DIST_STAGE) $(WINDOWS_INSTALL_SCRIPT)
	cp $(WINDOWS_INSTALL_SCRIPT) $@

# Produces the Windows upgrade ZIP archive for the release.
release-windows-upgrade-archive: release-prereqs
	$(MAKE) build-windows-upgrade-archive WINDOWS_ARCHIVE_TAG=$(VERSION)

# Build the Windows upgrade zip archive, which also builds the windows archive.
build-windows-upgrade-archive: clean-windows $(WINDOWS_UPGRADE_INSTALL_ZIP) $(WINDOWS_UPGRADE_INSTALL_FILE) $(WINDOWS_UPGRADE_SCRIPT) $(WINDOWS_UPGRADE_BUILD)
	rm $(WINDOWS_UPGRADE_ARCHIVE) || true
	cd $(WINDOWS_UPGRADE_DIST_STAGE) && zip -r "$(CURDIR)/$(WINDOWS_UPGRADE_ARCHIVE)" *.zip *.ps1

# Sets up the docker builder used to create Windows image tarballs.
setup-windows-builder:
	-docker buildx rm calico-windows-builder
	docker buildx create --name=calico-windows-builder --use --platform windows/amd64

# Builds all the Windows image tarballs for each version in WINDOWS_VERSIONS
image-tar-windows-all: setup-windows-builder $(addprefix sub-image-tar-windows-,$(WINDOWS_VERSIONS)) $(addprefix sub-image-tar-windows-upgrade-,$(WINDOWS_VERSIONS))

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

# Uses the docker builder to create a Windows image tarball for a single Windows version.
sub-image-tar-windows-%:
	# ensure dir for windows image tars
	-mkdir -p $(WINDOWS_DIST)
	# ensure docker build dir exists
	-mkdir -p $(WINDOWS_DIST_STAGE)
	# copy dockerfile to staging dir
	cp $(WINDOWS_PACKAGING_ROOT)/Dockerfile $(WINDOWS_DIST_STAGE)
	# copy install entrypoint script to staging dir
	cp $(WINDOWS_PACKAGING_ROOT)/host-process-install.ps1 $(WINDOWS_DIST_STAGE)
	# build and copy install script to staging dir
	make -C ../calico build
	cp ../calico/_site/scripts/install-calico-windows.ps1 $(WINDOWS_DIST_STAGE)
	# copy calico install zip file to staging dir
	cp dist/tigera-calico-windows-$(NODE_GIT_VERSION).zip $(WINDOWS_DIST_STAGE)/calico-windows.zip
	cd $(WINDOWS_DIST_STAGE) && \
	docker buildx build \
		--platform windows/amd64 \
		--output=type=docker,dest=$(CURDIR)/$(WINDOWS_DIST)/image-$(NODE_GIT_VERSION)-$*.tar \
		--pull \
		--build-arg=WINDOWS_VERSION=$* .

# Uses the docker builder to create a Windows upgrade image tarball for a single Windows version.
sub-image-tar-windows-upgrade-%:
	-mkdir -p $(WINDOWS_UPGRADE_DIST)
	cd $(WINDOWS_UPGRADE_ROOT) && \
		docker buildx build \
			--platform windows/amd64 \
			--output=type=docker,dest=$(CURDIR)/$(WINDOWS_UPGRADE_DIST)/image-$(NODE_GIT_VERSION)-$*.tar \
			--pull \
			--no-cache \
			--build-arg=WINDOWS_VERSION=$* .

# For EE, the images are:
# - calico-windows
# - calico-windows-upgrade
#
# The windows-upgrade image, for dev only, is also copied to the release tag version.
# The windows-upgrade procedure compares calico-node.exe's CNXRELEASEVERSION to the operator's windows-upgrade component image tag.
# That means the EE hash release must use a windows-upgrade image tag equal to the release version.
cd-windows-all:
	$(MAKE) cd-windows-calico-windows WINDOWS_IMAGE_TAR_ROOT=$(WINDOWS_DIST)
	$(MAKE) cd-windows-calico-windows-upgrade WINDOWS_IMAGE_TAR_ROOT=$(WINDOWS_UPGRADE_DIST)
	if [ "$(DEV_REGISTRIES)" = "gcr.io/unique-caldron-775/cnx/tigera" ]; then \
		dev_tag="gcr.io/unique-caldron-775/cnx/tigera/calico-windows-upgrade:$(NODE_GIT_VERSION)"; \
		dev_rel_tag="gcr.io/unique-caldron-775/cnx/tigera/calico-windows-upgrade:$(NODE_RELEASE_VERSION)"; \
		$(CRANE_BINDMOUNT) cp $${dev_tag} $${dev_rel_tag}$(double_quote) & \
	fi;

# Windows image pushing is different because we do not build docker images directly.
# Since the build machine is linux, we output the images to a tarball. (We can
# produce images but there will be no output because docker images
# built for Windows cannot be loaded on linux.)
#
# The resulting image tarball is then pushed to registries during cd/release.
# The image tarballs are located in WINDOWS_IMAGE_TAR_ROOT and have files names
# with the format 'image-v3.21.0-2-abcdef-20H2.tar'.
#
# In addition to pushing the individual images, we also create the manifest
# directly using 'docker manifest'. This is possible because Semaphore is using
# a recent enough docker CLI version (20.10.0)
#
# - Create the manifest with 'docker manifest create' using the list of all images.
# - For each windows version, 'docker manifest annotate' its image with "os.image: <windows_version>".
#   <windows_version> is the version string that looks like, e.g. 10.0.19041.1288.
#   Setting os.image in the manifest is required for Windows hosts to load the
#   correct image in manifest.
# - Finally we push the manifest, "purging" the local manifest.
cd-windows-%:
# WINDOWS_IMAGE_TAR_ROOT is the dir containing image tarballs to push.
ifndef WINDOWS_IMAGE_TAR_ROOT
	$(error WINDOWS_IMAGE_TAR_ROOT is not set)
endif
	for registry in $(DEV_REGISTRIES); do \
		echo Pushing Windows images to $${registry}; \
		all_images=""; \
		manifest_image="$${registry}/$*:$(NODE_GIT_VERSION)"; \
		for win_ver in $(WINDOWS_VERSIONS); do \
			image_tar="$(WINDOWS_IMAGE_TAR_ROOT)/image-$(NODE_GIT_VERSION)-$${win_ver}.tar"; \
			image="$${registry}/$*:$(NODE_GIT_VERSION)-windows-$${win_ver}"; \
			echo Pushing image $${image} ...; \
			$(CRANE_BINDMOUNT) push $${image_tar} $${image}$(double_quote) & \
			all_images="$${all_images} $${image}"; \
		done; \
		wait; \
		$(DOCKER_MANIFEST) create --amend $${manifest_image} $${all_images}; \
		for win_ver in $(WINDOWS_VERSIONS); do \
			version=$$(docker manifest inspect mcr.microsoft.com/windows/nanoserver:$${win_ver} | grep "os.version" | head -n 1 | awk -F\" '{print $$4}'); \
			image="$${registry}/$*:$(NODE_GIT_VERSION)-windows-$${win_ver}"; \
			$(DOCKER_MANIFEST) annotate --os windows --arch amd64 --os-version $${version} $${manifest_image} $${image}; \
		done; \
		$(DOCKER_MANIFEST) push --purge $${manifest_image}; \
	done ;
