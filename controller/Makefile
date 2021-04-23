##############################################################################
# Copyright 2019-20 Tigera Inc. All rights reserved.
##############################################################################
PACKAGE_NAME   ?= github.com/tigera/intrusion-detection/controller
GO_BUILD_VER   ?= v0.51
GIT_USE_SSH     = true
LIBCALICO_REPO  = github.com/tigera/libcalico-go-private
BUILD_IMAGE    ?= tigera/intrusion-detection-controller

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_INTRUSION_DETECTION_PROJECT_ID)

build: $(BUILD_IMAGE)

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"


# Figure out the GID of the docker group so that we can set the user inside the
# container to be a member of that group. This, combined with mounting the
# docker socket, allows the UTs to start docker containers.
MY_DOCKER_GID=$(shell getent group docker | cut -d: -f3)

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/* -e EXTRA_GROUP_ID=$(MY_DOCKER_GID) -v /var/run/docker.sock:/var/run/docker.sock:rw

PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=quay.io/$(BUILD_IMAGE)

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))


include Makefile.common

###############################################################################

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

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

###############################################################################
# Define some constants
###############################################################################
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
UNIT_TEST_FLAGS= -run $(UNIT_TESTS) -v
endif

CONTROLLER_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
CONTROLLER_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CONTROLLER_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CONTROLLER_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(CONTROLLER_VERSION) \
	-X main.BUILD_DATE=$(CONTROLLER_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(CONTROLLER_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(CONTROLLER_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

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
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/controller" && \
               ( ldd $(BINDIR)/controller-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
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
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/healthz" && \
               ( ldd $(BINDIR)/healthz-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
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

##########################################################################
# Testing
##########################################################################
.PHONY: ut
ut:
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv st
fv:
	echo "FV not implemented yet"
st:
	echo "ST not implemented yet"

.PHONY: clean
clean: clean-bin clean-build-image
	rm -rf vendor Makefile.common*
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
ci: clean test image-all

## Deploy images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(BRANCH_NAME) EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long --abbrev=12) EXCLUDEARCH="$(EXCLUDEARCH)"

##############################################################################
# Release
##############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty --abbrev=12)
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

###############################################################################
# Updating pins
###############################################################################
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

APISERVER_REPO=github.com/tigera/apiserver
APIMACHINERY_BRANCH=$(PIN_BRANCH)
LICENSING_BRANCH=$(PIN_BRANCH)
LICENSING_REPO=github.com/tigera/licensing

replace-licensing-pin:
	$(call update_replace_pin,$(LICENSING_REPO),$(LICENSING_REPO),$(LICENSING_BRANCH))

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin replace-apiserver-pin replace-licensing-pin
