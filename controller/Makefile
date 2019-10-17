# Copyright 2019 Tigera Inc. All rights reserved.

###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst docker-image/Dockerfile.%,%,$(wildcard docker-image/Dockerfile.*))

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

###############################################################################
BUILD_IMAGE?=tigera/intrusion-detection-controller
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=quay.io/$(BUILD_IMAGE)
PACKAGE_NAME?=github.com/tigera/intrusion-detection/controller

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

GO_BUILD_VER?=v0.26
# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
CALICO_BUILD=calico/go-build:$(GO_BUILD_VER)

help:
	@echo "Tigera Intrusion Detection Controller Makefile"
	@echo "Builds:"
	@echo
	@echo "  make all                                   Build all the binary packages."
	@echo "  make tigera/intrusion-detection-contoller  Build $(BUILD_IMAGE) docker image."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make test                Run Tests."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make clean         Remove binary files."
# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

all: $(BUILD_IMAGE)
test: ut fv

# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Define some constants
#######################
K8S_VERSION    = v1.10.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

CONTROLLER_VERSION?=$(shell git describe --tags --dirty --always)
CONTROLLER_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CONTROLLER_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CONTROLLER_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(CONTROLLER_VERSION) \
	-X main.BUILD_DATE=$(CONTROLLER_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(CONTROLLER_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(CONTROLLER_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

# Figure out the GID of the docker group so that we can set the user inside the
# container to be a member of that group. This, combined with mounting the
# docker socket, allows the UTs to start docker containers.
MY_DOCKER_GID=$(shell getent group docker | cut -d: -f3)

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

# A note on GOCACHE: the default directory for the build cache is
# `/home/user/.cache/go-build`. However, if we ask Docker to volume mount this
# directory, then the parent directory `/home/user` gets owned by root, meaning
# our later attempts to set gitconfig will fail. So, we mount the cache in
# `/go-build` and set the GOCACHE environment variable to use it.
DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   mkdir -p .go-build-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -e GOARCH=$(ARCH) \
                              -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
                              -v $${PWD}/../:/intrusion-detection/:rw \
                              -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
                              -v $${PWD}/.go-build-cache:/go-build:rw \
                              -e GOCACHE=/go-build \
                              -e GOSUMDB=off \
                              -v /var/run/docker.sock:/var/run/docker.sock:rw \
                              -e EXTRA_GROUP_ID=$(MY_DOCKER_GID) \
                              -w /intrusion-detection/controller \
                              $(CALICO_BUILD)


# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "controller" instead of "$(BINDIR)/controller".
#########################################################################
controller: $(BINDIR)/controller

$(BINDIR)/controller: $(BINDIR)/controller-amd64
	cd $(BINDIR) && (rm controller || ln -s -T controller-$(ARCH) controller)

$(BINDIR)/controller-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building controller...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	           go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/controller" && \
               ( ldd $(BINDIR)/controller-$(ARCH) 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $(BINDIR)/controller-$(ARCH) was not statically linked"; false ) )'

healthz: $(BINDIR)/healthz

$(BINDIR)/healthz: $(BINDIR)/healthz-amd64
	cd $(BINDIR) && (rm healthz || ln -s -T healthz-$(ARCH) healthz)

$(BINDIR)/healthz-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building healthz...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	           go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/healthz" && \
               ( ldd $(BINDIR)/healthz-$(ARCH) 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $(BINDIR)/healthz-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(BUILD_IMAGE) $(BUILD_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(BUILD_IMAGE)
$(BUILD_IMAGE): $(BUILD_IMAGE)-$(ARCH)
$(BUILD_IMAGE)-$(ARCH): $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp $(BINDIR)/controller-$(ARCH) docker-image/bin/
	cp $(BINDIR)/healthz-$(ARCH) docker-image/bin/
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif

imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))

sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

## push all arches
push-all: imagetag $(addprefix sub-push-,$(ARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
	# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(ARCHES)) --template $(call unescapefs,$*:$(IMAGETAG))-ARCH --target $(call unescapefs,$*:$(IMAGETAG))"

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
tag-images-all: imagetag $(addprefix sub-tag-images-,$(ARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

##########################################################################
# Testing
##########################################################################
.PHONY: ut
ut:
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
		    go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv:
	echo "FV not implemented yet"

.PHONY: clean
clean: clean-bin clean-build-image
clean-build-image:
	docker rmi -f $(BUILD_IMAGE) > /dev/null 2>&1 || true

clean-bin:
	rm -rf $(BINDIR) \
			docker-image/bin

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
ci: test image-all

## Deploy images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(BRANCH_NAME) EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

##############################################################################
# Release
##############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty)
ifndef VERSION
	BUILD_VERSION = $(GIT_VERSION)
else
	BUILD_VERSION = $(VERSION)
endif


## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif

	$(MAKE) image-all 
	$(MAKE) tag-images-all IMAGETAG=$(VERSION) RELEASE=true
	# Generate the `latest` images.
	$(MAKE) tag-images-all IMAGETAG=latest RELEASE=true

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	for img in quay.io/$(BUILD_IMAGE):$(VERSION)-$(ARCH); do \
	  if docker run $$img --version | grep -q '$(VERSION)$$'; \
	  then \
	    echo "Check successful. ($$img)"; \
	  else \
	    echo "Incorrect version in docker image $$img!"; \
	    exit 1; \
	  fi \
	done; \

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif


# Run fossa.io license checks
foss-checks:
	@echo Running $@...
	$(DOCKER_GO_BUILD) \
			sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
					/usr/local/bin/fossa'
