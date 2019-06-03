# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut fv st

###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst Dockerfile.%,%,$(wildcard Dockerfile.*))

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)
BUILDOS ?= $(shell uname -s | tr A-Z a-z)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
        BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
        BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
    override ARCH=amd64
endif

# Build mounts for running in "local build" mode. Mount in libcalico, confd, and felix but null out
# their respective vendor directories. This allows an easy build of calico/node using local development code,
# assuming that there is a local checkout of felix, confd, and libcalico in the same directory as the node repo.
LOCAL_BUILD_MOUNTS ?=
ifeq ($(LOCAL_BUILD),true)
LOCAL_BUILD_MOUNTS = -v $(CURDIR)/../libcalico-go:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go/vendor:ro \
	-v $(CURDIR)/../confd:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/confd:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/confd/vendor:ro \
	-v $(CURDIR)/../felix:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/felix:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/felix/vendor:ro
endif

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

# Targets used when cross building.
.PHONY: register
# Enable binfmt adding support for miscellaneous binary formats.
# This is only needed when running non-native binaries.
register:
ifneq ($(BUILDARCH),$(ARCH))
	docker run --rm --privileged multiarch/qemu-user-static:register || true
endif

# list of arches *not* to build when doing *-all
#    until s390x works correctly
EXCLUDEARCH ?= s390x
VALIDARCHES = $(filter-out $(EXCLUDEARCH),$(ARCHES))

###############################################################################
CNX_REPOSITORY?=gcr.io/unique-caldron-775/cnx
BUILD_IMAGE?=tigera/cnx-node
PUSH_IMAGES?=$(CNX_REPOSITORY)/tigera/cnx-node
RELEASE_IMAGES?=

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

LOCAL_USER_ID?=$(shell id -u $$USER)

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

GO_BUILD_VER?=v0.20
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)

#This is a version with known container with compatible versions of sed/grep etc. 
TOOLING_BUILD?=calico/go-build:v0.20	


# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

DOCKER_RUN := mkdir -p .go-pkg-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                              -v $(HOME)/.glide:/home/user/.glide:rw \
                              -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
                              -v $(CURDIR)/.go-pkg-cache:/go/pkg:rw \
                              -w /go/src/$(PACKAGE_NAME) \
                              -e GOARCH=$(ARCH)


# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

# Version of this repository as reported by git.
CALICO_GIT_VER=v3.6.0
CNX_GIT_VER := $(shell git describe --tags --dirty --always)
ifeq ($(LOCAL_BUILD),true)
	CNX_GIT_VER = $(shell git describe --tags --dirty --always)-dev-build
endif

# Versions and location of dependencies used in the build.
BIRD_VER?=v0.3.3-0-g1e8dd375
BIRD_IMAGE ?= calico/bird:$(BIRD_VER)-$(ARCH)

# Versions and locations of dependencies used in tests.
CALICOCTL_VER?=master
CNI_VER?=master
TEST_CONTAINER_NAME_VER?=latest
CTL_CONTAINER_NAME?=$(CNX_REPOSITORY)/tigera/calicoctl:$(CALICOCTL_VER)-$(ARCH)
TEST_CONTAINER_NAME?=calico/test:$(TEST_CONTAINER_NAME_VER)-$(ARCH)
ETCD_VERSION?=v3.3.7
# If building on amd64 omit the arch in the container name.  Fixme!
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)
ifneq ($(BUILDARCH),amd64)
        ETCD_IMAGE=$(ETCD_IMAGE)-$(ARCH)
endif

K8S_VERSION?=v1.11.3
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
    vendor/github.com/kelseyhightower/confd/windows-packaging/config-bgp.ps1 \
    vendor/github.com/kelseyhightower/confd/windows-packaging/config-bgp.psm1 \
    vendor/github.com/kelseyhightower/confd/windows-packaging/conf.d/blocks.toml \
    vendor/github.com/kelseyhightower/confd/windows-packaging/conf.d/peerings.toml \
    vendor/github.com/kelseyhightower/confd/windows-packaging/templates/blocks.ps1.template \
    vendor/github.com/kelseyhightower/confd/windows-packaging/templates/peerings.ps1.template \
    vendor/github.com/kelseyhightower/confd/windows-packaging/config-bgp.ps1 \
    vendor/github.com/kelseyhightower/confd/windows-packaging/config-bgp.psm1 \
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
CRD_PATH=$(CURDIR)/vendor/github.com/projectcalico/libcalico-go/test/
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')
ST_TO_RUN?=tests/st/
K8ST_TO_RUN?=tests/
# Can exclude the slower tests with "-a '!slow'"
ST_OPTIONS?=

# Variables for building the local binaries that go into the image
MAKE_SURE_BIN_EXIST := $(shell mkdir -p dist .go-pkg-cache $(NODE_CONTAINER_BIN_DIR))
NODE_CONTAINER_FILES=$(shell find ./filesystem -type f)
SRCFILES=$(shell find ./pkg -name '*.go')
LDFLAGS=-ldflags "-X github.com/projectcalico/node/pkg/startup.CNXVERSION=$(CNX_GIT_VER) -X github.com/projectcalico/node/pkg/startup.CALICOVERSION=$(CALICO_GIT_VER) \
                  -X main.VERSION=$(CALICO_GIT_VER) \
                  -X github.com/projectcalico/node/vendor/github.com/projectcalico/felix/buildinfo.GitVersion=$(CALICO_GIT_VER) \
                  -X github.com/projectcalico/node/vendor/github.com/projectcalico/felix/buildinfo.GitRevision=$(shell git rev-parse HEAD || echo '<unknown>')"
PACKAGE_NAME?=github.com/projectcalico/node
LIBCALICOGO_PATH?=none

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

## Clean enough that a new release build will be clean
clean:
	find . -name '*.created' -exec rm -f {} +
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf certs *.tar vendor $(NODE_CONTAINER_BIN_DIR)
	rm -f $(WINDOWS_ARCHIVE_BINARY) $(WINDOWS_BINARY)
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/confd/templates/*
	rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1
	rm -f $(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt
	# Delete images that we built in this repo
	docker rmi $(BUILD_IMAGE):latest-$(ARCH) || true
	docker rmi $(TEST_CONTAINER_NAME) || true

###############################################################################
# Building the binary
###############################################################################
build:  $(NODE_CONTAINER_BINARY)
# Use this to populate the vendor directory after checking out the repository.
# To update upstream dependencies, delete the glide.lock file first.
vendor: glide.lock
	# Ensure that the glide cache directory exists.
	mkdir -p $(HOME)/.glide

	# To build without Docker just run "glide install -strip-vendor"
	if [ "$(LIBCALICOGO_PATH)" != "none" ]; then \
          EXTRA_DOCKER_BIND="-v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro"; \
	fi; \

	docker run --rm \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw $$EXTRA_DOCKER_BIND \
		-v $(HOME)/.glide:/home/user/.glide:rw \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) glide install -strip-vendor


$(NODE_CONTAINER_BINARY): vendor $(SRCFILES)
	docker run --rm \
		-e GOARCH=$(ARCH) \
		-e GOOS=linux \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
		-e GOCACHE=/go-cache \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) go build -v -o $@ $(LDFLAGS) ./cmd/calico-node/main.go

$(WINDOWS_BINARY): vendor
	docker run --rm \
		-e GOARCH=$(ARCH) \
		-e GOOS=windows \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
		-e GOCACHE=/go-cache \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) go build -v -o $@ $(LDFLAGS) ./cmd/calico-node/main.go

$(WINDOWS_ARCHIVE_ROOT)/cni/calico.exe: glide.lock vendor
	docker run --rm \
		-e GOARCH=$(ARCH) \
		-e GOOS=windows \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
		-e GOCACHE=/go-cache \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) go build -v -o $@ $(LDFLAGS) ./cmd/calico

$(WINDOWS_ARCHIVE_ROOT)/cni/calico-ipam.exe: glide.lock vendor
	docker run --rm \
		-e GOARCH=$(ARCH) \
		-e GOOS=windows \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
		-e GOCACHE=/go-cache \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) go build -v -o $@ $(LDFLAGS) ./cmd/calico-ipam

###############################################################################
# Building the image
###############################################################################
## Create the image for the current ARCH
image: $(BUILD_IMAGE)
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(BUILD_IMAGE): $(NODE_CONTAINER_CREATED)
$(NODE_CONTAINER_CREATED): ./Dockerfile.$(ARCH) $(NODE_CONTAINER_FILES) $(NODE_CONTAINER_BINARY)
	$(MAKE) register
	# Check versions of the binaries that we're going to use to build the image.
	# Since the binaries are built for Linux, run them in a container to allow the
	# make target to be run on different platforms (e.g. MacOS).
	docker run --rm -v $(CURDIR)/dist/bin:/go/bin:rw $(CALICO_BUILD) /bin/sh -c "\
	  echo; echo calico-node-$(ARCH) -v;         /go/bin/calico-node-$(ARCH) -v; \
	"
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) . --build-arg BIRD_IMAGE=$(BIRD_IMAGE) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg ver=$(CALICO_GIT_VER) -f ./Dockerfile.$(ARCH)
	touch $@

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))

sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

## push all supported arches
push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
	# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
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


## tag version number build images i.e.  tigera/node:latest-amd64 -> tigera/node:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)



###############################################################################
# Windows packaging
###############################################################################

# Pull the BGP configuration scripts and templates from the confd repo.
$(WINDOWS_VENDORED_FILES): vendor

$(WINDOWS_ARCHIVE_ROOT)/confd/config-bgp%: vendor/github.com/kelseyhightower/confd/windows-packaging/config-bgp%
	cp $< $@

$(WINDOWS_ARCHIVE_ROOT)/confd/conf.d/%: vendor/github.com/kelseyhightower/confd/windows-packaging/conf.d/%
	cp $< $@

$(WINDOWS_ARCHIVE_ROOT)/confd/templates/%: vendor/github.com/kelseyhightower/confd/windows-packaging/templates/%
	cp $< $@

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/hns.psm1: ./vendor/github.com/Microsoft/SDN/Kubernetes/windows/hns.psm1
	cp $< $@

$(WINDOWS_ARCHIVE_ROOT)/libs/hns/License.txt: ./vendor/github.com/Microsoft/SDN/License.txt
	cp $< $@

## Download NSSM.
windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip:
	wget -O windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip https://nssm.cc/release/nssm-$(WINDOWS_NSSM_VERSION).zip

build-windows-archive: $(WINDOWS_ARCHIVE_FILES) windows-packaging/nssm-$(WINDOWS_NSSM_VERSION).zip
	# To be as atomic as possible, we re-do work like unpacking NSSM here.
	-rm -f "$(WINDOWS_ARCHIVE)"
	-rm -rf $(WINDOWS_ARCHIVE_ROOT)/nssm-$(WINDOWS_NSSM_VERSION)
	mkdir -p dist
	cd windows-packaging && \
	sha256sum --check nssm.sha256sum && \
	cd TigeraCalico && \
	unzip  ../nssm-$(WINDOWS_NSSM_VERSION).zip \
	       -x 'nssm-$(WINDOWS_NSSM_VERSION)/src/*' && \
	cd .. && \
	zip -r "../$(WINDOWS_ARCHIVE)" TigeraCalico -x '*.git*'
	@echo
	@echo "Windows archive built at $(WINDOWS_ARCHIVE)"

$(WINDOWS_ARCHIVE_BINARY): $(WINDOWS_BINARY)
	cp $< $@



###############################################################################
# Managing the upstream library pins
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-licensing-pin update-libcalico-pin update-felix-pin update-cni-plugin-pin update-confd-pin

## deprecated target alias
update-libcalico: update-pins
	$(warning !! Update update-libcalico is deprecated, use update-pins !!)

update-felix-confd-libcalico: update-pins
	$(warning !! Update update-felix-confd-libcalico is deprecated, use update-pins !!)

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;


###############################################################################
## libcalico

## Set the default LIBCALICO source for this project
LIBCALICO_PROJECT_DEFAULT=tigera/libcalico-go-private.git
LIBCALICO_GLIDE_LABEL=projectcalico/libcalico-go

## Default the LIBCALICO repo and version but allow them to be overridden (master or release-vX.Y)
## default LIBCALICO branch to the same branch name as the current checked out repo
LIBCALICO_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LIBCALICO_REPO?=github.com/$(LIBCALICO_PROJECT_DEFAULT)
LIBCALICO_VERSION?=$(shell git ls-remote git@github.com:$(LIBCALICO_PROJECT_DEFAULT) $(LIBCALICO_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure LIBCALICO repo and branch are reachable
guard-git-libcalico: 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ; 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Ensure the branch exists, or set LIBCALICO_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(LIBCALICO_VERSION))" = "" ]; then \
		echo "ERROR: LIBCALICO version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-libcalico-pin: guard-ssh-forwarding-bug guard-git-libcalico
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(LIBCALICO_GLIDE_LABEL)" \
		REPO="$(LIBCALICO_REPO)" \
		VERSION="$(LIBCALICO_VERSION)" \
		DEFAULT_REPO="$(LIBCALICO_PROJECT_DEFAULT)" \
		BRANCH="$(LIBCALICO_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '

###############################################################################
## felix

## Set the default FELIX source for this project 
FELIX_PROJECT_DEFAULT=tigera/felix-private.git
FELIX_GLIDE_LABEL=projectcalico/felix

## Default the FELIX repo and version but allow them to be overridden (master or release-vX.Y)
## default FELIX branch to the same branch name as the current checked out repo
FELIX_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
FELIX_REPO?=github.com/$(FELIX_PROJECT_DEFAULT)
FELIX_VERSION?=$(shell git ls-remote git@github.com:$(FELIX_PROJECT_DEFAULT) $(FELIX_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure FELIX repo and branch are reachable
guard-git-felix: 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ; 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "$(FELIX_BRANCH)" "Ensure the branch exists, or set FELIX_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "$(FELIX_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(FELIX_VERSION))" = "" ]; then \
		echo "ERROR: FELIX version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-felix-pin: guard-ssh-forwarding-bug guard-git-felix
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(FELIX_GLIDE_LABEL)" \
		REPO="$(FELIX_REPO)" \
		VERSION="$(FELIX_VERSION)" \
		DEFAULT_REPO="$(FELIX_PROJECT_DEFAULT)" \
		BRANCH="$(FELIX_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '


###############################################################################
## cni-plugin

## Set the default CNIPLUGIN source for this project 
CNIPLUGIN_PROJECT_DEFAULT=tigera/cni-plugin-private.git
CNIPLUGIN_GLIDE_LABEL=projectcalico/cni-plugin

## Default the CNIPLUGIN repo and version but allow them to be overridden (master or release-vX.Y)
## default CNIPLUGIN branch to the same branch name as the current checked out repo
CNIPLUGIN_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
CNIPLUGIN_REPO?=github.com/$(CNIPLUGIN_PROJECT_DEFAULT)
CNIPLUGIN_VERSION?=$(shell git ls-remote git@github.com:$(CNIPLUGIN_PROJECT_DEFAULT) $(CNIPLUGIN_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure CNIPLUGIN repo and branch are reachable
guard-git-cni-plugin: 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(CNIPLUGIN_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ; 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(CNIPLUGIN_PROJECT_DEFAULT) "$(CNIPLUGIN_BRANCH)" "Ensure the branch exists, or set CNIPLUGIN_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(CNIPLUGIN_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(CNIPLUGIN_PROJECT_DEFAULT) "$(CNIPLUGIN_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(CNIPLUGIN_VERSION))" = "" ]; then \
		echo "ERROR: CNIPLUGIN version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-cni-plugin-pin: guard-ssh-forwarding-bug guard-git-cni-plugin
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(CNIPLUGIN_GLIDE_LABEL)" \
		REPO="$(CNIPLUGIN_REPO)" \
		VERSION="$(CNIPLUGIN_VERSION)" \
		DEFAULT_REPO="$(CNIPLUGIN_PROJECT_DEFAULT)" \
		BRANCH="$(CNIPLUGIN_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '


###############################################################################
## licensing

## Set the default LICENSING source for this project 
LICENSING_PROJECT_DEFAULT=tigera/licensing
LICENSING_GLIDE_LABEL=tigera/licensing

## Default the LICENSING repo and version but allow them to be overridden (master or release-vX.Y)
## default LICENSING branch to the same branch name as the current checked out repo
LICENSING_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LICENSING_REPO?=github.com/$(LICENSING_PROJECT_DEFAULT)
LICENSING_VERSION?=$(shell git ls-remote git@github.com:$(LICENSING_PROJECT_DEFAULT) $(LICENSING_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure LICENSING repo and branch are reachable
guard-git-licensing: 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ; 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "$(LICENSING_BRANCH)" "Ensure the branch exists, or set LICENSING_BRANCH variable";
	@$(DOCKER_RUN) $(TOOLING_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "$(LICENSING_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(LICENSING_VERSION))" = "" ]; then \
		echo "ERROR: LICENSING version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-licensing-pin: guard-ssh-forwarding-bug guard-git-licensing
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(LICENSING_GLIDE_LABEL)" \
		REPO="$(LICENSING_REPO)" \
		VERSION="$(LICENSING_VERSION)" \
		DEFAULT_REPO="$(LICENSING_PROJECT_DEFAULT)" \
		BRANCH="$(LICENSING_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '


###############################################################################
## confd

## Set the default CONFD source for this project 
CONFD_PROJECT_DEFAULT=tigera/confd-private.git
CONFD_GLIDE_LABEL=projectcalico/confd

## Default the CONFD repo and version but allow them to be overridden (master or release-vX.Y)
## default CONFD branch to the same branch name as the current checked out repo
CONFD_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
CONFD_REPO?=github.com/$(CONFD_PROJECT_DEFAULT)
CONFD_VERSION?=$(shell git ls-remote git@github.com:$(CONFD_PROJECT_DEFAULT) $(CONFD_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure CONFD repo and branch are reachable
guard-git-confd: 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(CONFD_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ; 
	@_scripts/functions.sh ensure_can_reach_repo_branch $(CONFD_PROJECT_DEFAULT) "$(CONFD_BRANCH)" "Ensure the branch exists, or set CONFD_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(CONFD_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(CONFD_PROJECT_DEFAULT) "$(CONFD_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(CONFD_VERSION))" = "" ]; then \
		echo "ERROR: CONFD version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-confd-pin: guard-ssh-forwarding-bug guard-git-confd
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(CONFD_GLIDE_LABEL)" \
		REPO="$(CONFD_REPO)" \
		VERSION="$(CONFD_VERSION)" \
		DEFAULT_REPO="$(CONFD_PROJECT_DEFAULT)" \
		BRANCH="$(CONFD_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '



###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
## Perform static checks on the code.
static-checks: vendor
	docker run --rm \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) gometalinter --deadline=300s --disable-all --enable=vet --enable=errcheck --enable=goimports --vendor pkg/...

.PHONY: fix
## Fix static checks
fix:
	goimports -w $(SRCFILES)

foss-checks: vendor
	@echo Running $@...
	@docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(CALICO_BUILD) /usr/local/bin/fossa

###############################################################################
# Unit tests
###############################################################################
## Run the ginkgo UTs.
ut: vendor
	docker run --rm \
	-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	$(LOCAL_BUILD_MOUNTS) \
	-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	--net=host \
	-w /go/src/$(PACKAGE_NAME) \
	$(CALICO_BUILD) ginkgo -cover -r cmd/calico $(GINKGO_ARGS)

###############################################################################
# FV Tests
###############################################################################
## Run the ginkgo FVs
fv: vendor run-k8s-apiserver
	docker run --rm \
	-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	$(LOCAL_BUILD_MOUNTS) \
	-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	-e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 \
	--net=host \
	-w /go/src/$(PACKAGE_NAME) \
	$(CALICO_BUILD) ginkgo -cover -r -skipPackage vendor pkg/startup pkg/allocateipip $(GINKGO_ARGS)

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
run-k8s-apiserver: stop-k8s-apiserver run-etcd
	docker run \
		--net=host --name st-apiserver \
		-v  $(CRD_PATH):/manifests \
		--detach \
		${HYPERKUBE_IMAGE} \
		/hyperkube apiserver \
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
	# from the manifest crds.yaml
	while ! docker exec st-apiserver kubectl \
		apply -f /manifests/crds.yaml; \
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
	-docker rm -f calicoctl
	docker pull $(CTL_CONTAINER_NAME)
	docker create --name calicoctl $(CTL_CONTAINER_NAME)
	docker cp calicoctl:calicoctl dist/calicoctl && \
	  test -e dist/calicoctl && \
	  touch dist/calicoctl
	-docker rm -f calicoctl

dist/calico-cni-plugin dist/calico-ipam-plugin:
	-docker rm -f calico-cni
	docker pull $(CNX_REPOSITORY)/tigera/cni:$(CNI_VER)
	docker create --name calico-cni $(CNX_REPOSITORY)/tigera/cni:$(CNI_VER)
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
	  echo bird --version;         /bin/bird --version; \
	"
	docker save --output $@ $(BUILD_IMAGE):latest-$(ARCH)

.PHONY: st-checks
st-checks:
	# Check that we're running as root.
	test `id -u` -eq '0' || { echo "STs must be run as root to allow writes to /proc"; false; }

	# Insert an iptables rule to allow access from our test containers to etcd
	# running on the host.
	iptables-save | grep -q 'calico-st-allow-etcd' || iptables $(IPT_ALLOW_ETCD)

## Get the kubeadm-dind-cluster script
K8ST_VERSION?=v1.12
DIND_SCR?=dind-cluster-$(K8ST_VERSION).sh
GCR_IO_PULL_SECRET?=${HOME}/gcr-pull-secret.json
TSEE_TEST_LICENSE?=${HOME}/new-test-customer-license.yaml

.PHONY: k8s-test
## Run the k8s tests
k8s-test:
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
	    sh -c 'cp /root/.kubeadm-dind-cluster/kubectl /bin/kubectl && cd /code/tests/k8st && \
	           nosetests $(K8ST_TO_RUN) -v --with-xunit --xunit-file="/code/report/k8s-tests.xml" --with-timer'

# Needed for Semaphore CI (where disk space is a real issue during k8s-test)
.PHONY: remove-go-build-image
remove-go-build-image:
	@echo "Removing $(CALICO_BUILD) image to save space needed for testing ..."
	@-docker rmi $(CALICO_BUILD)

.PHONY: st
## Run the system tests
st: dist/calicoctl busybox.tar cnx-node.tar workload.tar run-etcd calico_test.created dist/calico-cni-plugin dist/calico-ipam-plugin
	# Check versions of Calico binaries that ST execution will use.
	docker run --rm -v $(CURDIR)/dist:/go/bin:rw $(CALICO_BUILD) /bin/sh -c "\
	  echo; echo calicoctl --version;        /go/bin/calicoctl --version; \
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
## Run what CI runs
ci: static-checks ut fv image-all build-windows-archive st

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
GIT_VERSION?=$(shell git describe --tags --dirty)
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
	if ! docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) versions | grep "^$(VERSION)$$"; then echo "Reported version:" `docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) versions` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

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
# Utilities
###############################################################################
.PHONY: help
## Display this help text
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	$(info Available targets)
	@awk '/^[a-zA-Z\-\_0-9\/]+:/ {                                      \
		nb = sub( /^## /, "", helpMsg );                                \
		if(nb == 0) {                                                   \
			helpMsg = $$0;                                              \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                   \
		}                                                               \
		if (nb)                                                         \
			printf "\033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;  \
	}                                                                   \
	{ helpMsg = $$0 }'                                                  \
	width=20                                                            \
	$(MAKEFILE_LIST)

$(info "Build dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "BIRD_VER" $(BIRD_VER)))

$(info "Test dependency versions")
$(info $(shell printf "%-21s = %-10s\n" "CNI_VER" $(CNI_VER)))

$(info "Calico git version")
$(info $(shell printf "%-21s = %-10s\n" "CALICO_GIT_VER" $(CALICO_GIT_VER)))
