# Copyright 2019-2024 Tigera Inc. All rights reserved.

GO_BUILD_VER ?= v0.90

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
ARCHES=windows-1809 windows-2022
else
# For Windows we append to the image tag to identify the Windows 10 version.
# For example, "v3.5.0-calient-0.dev-26-gbaba2f0b96a4-windows-1903"
#
# We support these platforms:
# - Windows 10 1809 amd64
# - Windows 10 2022 amd64
#
# For Linux, we leave the image tag alone.
ifeq ($(OS),Windows_NT)
FLUENTD_IMAGE ?= tigera/fluentd-windows

# Get the Windows build number.
$(eval WINDOWS_BUILD_VERSION := $(shell (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuild))

# Get Windows version based on build number.
ifeq ($(WINDOWS_BUILD_VERSION),17763)
WINDOWS_VERSION := 1809
else ifeq ($(WINDOWS_BUILD_VERSION),20348)
WINDOWS_VERSION := 2022
else
$(error Unknown WINDOWS_BUILD_VERSION)
endif

ARCHES ?=windows-$(WINDOWS_VERSION)
else
FLUENTD_IMAGE ?= tigera/fluentd
ARCHES        ?= amd64 arm64
endif
endif

BUILD_IMAGES          ?= $(FLUENTD_IMAGE)
DEV_REGISTRIES        ?= gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?= quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

# This variable is used to filter out values from DEV_REGISTRIES, with the
# result assigned to MANIFEST_REGISTRIES.
# Currently, go-build v0.65 sets this to empty but that results in an empty
# MANIFEST_REGISTRIES so we set NONMANIFEST_REGISTRIES to a placeholder value for now.
NONMANIFEST_REGISTRIES ?= non-manifest-registries-defined-here

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
ifeq ($(OS),Windows_NT)
	rm -force Makefile.common.*
	curl -usebasicparsing -uri $(MAKE_REPO)/Makefile.common -outfile "$@"
else
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"
endif

include Makefile.common

###############################################################################
# Build
###############################################################################
.PHONY: build
build: eks-log-forwarder-startup

## build cloudwatch plugin initializer
eks-log-forwarder-startup:
	$(MAKE) -C eks/ eks-log-forwarder-startup

clean:
	rm -rf eks/bin Makefile.common*
	rm -f $(FLUENTD_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(FLUENTD_IMAGE) -a -q)

###############################################################################
# Image
###############################################################################
FLUENTD_IMAGE_CREATED=.fluentd.created-$(ARCH)

QEMU_IMAGE ?= calico/qemu-user-static:latest

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES)) 
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(FLUENTD_IMAGE)

$(FLUENTD_IMAGE): $(FLUENTD_IMAGE_CREATED)
$(FLUENTD_IMAGE_CREATED): register build
ifeq ($(OS),Windows_NT)
	docker build --pull -t $(FLUENTD_IMAGE):latest-amd64 -f Dockerfile.windows .
else
	docker buildx build --load --platform=linux/$(ARCH) --pull \
		--build-arg QEMU_IMAGE=$(QEMU_IMAGE) \
		-t $(FLUENTD_IMAGE):latest-$(ARCH) -f Dockerfile .
endif
ifeq ($(ARCH),amd64)
	docker tag $(FLUENTD_IMAGE):latest-$(ARCH) $(FLUENTD_IMAGE):latest
endif
	touch $@

###############################################################################
# CI/CD
###############################################################################
## test cloudwatch plugin initializer
test-eks-log-forwarder-startup: eks-log-forwarder-startup
	$(MAKE) -C eks/ ut

ut:
st:

## fluentd config tests
fv: image eks-log-forwarder-startup
ifeq ($(ARCH),amd64)
	cd test && IMAGETAG=latest ./test.sh
	$(MAKE) -C eks/ ut
endif

ci: image test

## push fluentd image to GCR_REPO.
#  Note: this is called from both Linux and Windows so ARCH_TAG is required.
cd: image-all cd-common

# create fluentd windows manifests
NANOSERVER_VERSIONS ?= 1809 ltsc2022
push-windows-manifest: var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME  $(addprefix sub-windows-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-windows-manifest-%:
	for imagetag in $(BRANCH_NAME) $(GIT_VERSION); do \
		docker manifest create $(call unescapefs,$*):$${imagetag} $(addprefix --amend ,$(addprefix $(call unescapefs,$*):$${imagetag}-,$(ARCHES))); \
		for win_ver in $(NANOSERVER_VERSIONS); do \
			ver=$$(docker manifest inspect mcr.microsoft.com/windows/nanoserver:$${win_ver} | jq -r '.manifests[0].platform."os.version"'); \
			image=$(call unescapefs,$*):$${imagetag}-windows-$$(printf '%s' $${win_ver} | sed 's/ltsc//g'); \
			docker manifest annotate --os windows --arch amd64 --os-version $${ver} $(call unescapefs,$*):$${imagetag} $${image}; \
		done; \
		docker manifest push --purge $(call unescapefs,$*):$${imagetag}; \
	done;
