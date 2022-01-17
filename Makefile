GO_BUILD_VER ?= v0.65

# Override shell if we're on Windows
# https://stackoverflow.com/a/63840549
ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command
endif

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_FLUENTD_DOCKER_PROJECT_ID)

# If this is a windows release we're not building the images and they will all be "cut" together
ifdef WINDOWS_RELEASE
FLUENTD_IMAGE=tigera/fluentd-windows
ARCHES=windows-1809 windows-20H2
else
# For Windows we append to the image tag to identify the Windows 10 version.
# For example, "v3.5.0-calient-0.dev-26-gbaba2f0b96a4-windows-1903"
#
# We support these platforms:
# - Windows 10 1809 amd64
# - Windows 10 20H2 amd64
#
# For Linux, we leave the image tag alone.
ifeq ($(OS),Windows_NT)
FLUENTD_IMAGE ?= tigera/fluentd-windows
DOCKERFILE    ?= Dockerfile.windows

# Get the Windows build number.
$(eval WINDOWS_BUILD_VERSION := $(shell (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuild))

# Get Windows version based on build number.
ifeq ($(WINDOWS_BUILD_VERSION),17763)
WINDOWS_VERSION := 1809
else ifeq ($(WINDOWS_BUILD_VERSION),19042)
WINDOWS_VERSION := 20H2
else
$(error Unknown WINDOWS_BUILD_VERSION)
endif

ARCHES ?=windows-$(WINDOWS_VERSION)
else
FLUENTD_IMAGE ?= tigera/fluentd
DOCKERFILE    ?= Dockerfile
ARCHES        ?= amd64
endif
endif

BUILD_IMAGES          ?= $(FLUENTD_IMAGE)
DEV_REGISTRIES        ?= gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?= quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev


GCR_REPO?=gcr.io/unique-caldron-775/cnx
REPO?=$(GCR_REPO)
PUSH_IMAGE_BASE?=$(REPO)
PUSH_IMAGE?=$(PUSH_IMAGE_BASE)/$(BUILD_IMAGE)

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

include Makefile.common

SRC_DIR?=$(PWD)
# Overwrite configuration, e.g. GCR_REPO:=gcr.io/tigera-dev/experimental/gaurav

## build cloudwatch plugin initializer
eks-log-forwarder-startup:
	$(MAKE) -C eks/ eks-log-forwarder-startup

build: eks-log-forwarder-startup

###############################################################################
# Build
###############################################################################

# Add --squash argument for CICD pipeline runs only to avoid setting "experimental",
# for Docker processes on personal machine.
# DOCKER_SQUASH is defaulted to be empty but can be set `DOCKER_SQUASH=--squash make image` 
# to squash images locally.
ifdef CI
DOCKER_SQUASH=--squash
endif

$(FLUENTD_IMAGE):
	$(MAKE) $(addprefix build-image-,$(VALIDARCHES)) IMAGE=$(FLUENTD_IMAGE) DOCKERFILE=$(DOCKERFILE)

build-image-%:
	docker build --pull $(DOCKER_SQUASH) -t $(IMAGE):latest-$* --file $(DOCKERFILE) .
	$(if $(filter amd64,$*),\
		docker tag $(IMAGE):latest-$* $(IMAGE):latest)

image: build $(FLUENTD_IMAGE)

clean-image: require-all-IMAGETAG
	-docker rmi $(FLUENT_IMAGE):latest-$(ARCH) $(FLUENT_IMAGE):latest

## clean slate cloudwatch plugin initializer
clean-eks-log-forwarder-startup:
	$(MAKE) -C eks/ clean

clean: clean-eks-log-forwarder-startup
	rm -rf Makefile.common*

## test cloudwatch plugin initializer
test-eks-log-forwarder-startup: eks-log-forwarder-startup
	$(MAKE) -C eks/ ut

ut:
st:

## fluentd config tests
fv: image eks-log-forwarder-startup
	cd $(SRC_DIR)/test && IMAGETAG=latest ./test.sh && cd $(SRC_DIR)
	$(MAKE) -C eks/ ut

ci: build test image

## push fluentd image to GCR_REPO.
#  Note: this is called from both Linux and Windows so ARCH_TAG is required.
cd: image cd-common

#
push-windows-manifest: var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME
	$(MAKE) push-manifests IMAGETAG=$(BRANCH_NAME) OUTPUT_DIR=/tmp/ MANIFEST_TOOL_SPEC_TEMPLATE=manifest-tool-spec.yaml.tpl.sh MANIFEST_TOOL_EXTRA_DOCKER_ARGS="-v /tmp:/tmp" FLUENTD_IMAGE=tigera/fluentd-windows
	$(MAKE) push-manifests IMAGETAG=$(shell git describe --tags --dirty --long --always --abbrev=12) OUTPUT_DIR=/tmp/ MANIFEST_TOOL_SPEC_TEMPLATE=manifest-tool-spec.yaml.tpl.sh MANIFEST_TOOL_EXTRA_DOCKER_ARGS="-v /tmp:/tmp" FLUENTD_IMAGE=tigera/fluentd-windows
