# Copyright 2019-2024 Tigera Inc. All rights reserved.

include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/fluentd

# Override shell if we're on Windows
# https://stackoverflow.com/a/63840549
ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command
endif

# If this is a windows release we're not building the images and they will all be "cut" together
ifdef WINDOWS_RELEASE
FLUENTD_IMAGE=fluentd-windows
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
FLUENTD_IMAGE ?= fluentd-windows

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

DEV_REGISTRIES = gcr.io/unique-caldron-775/cnx/tigera
else
FLUENTD_IMAGE ?= fluentd
ARCHES        ?= amd64 arm64
endif
endif

BUILD_IMAGES          ?= $(FLUENTD_IMAGE)

##############################################################################
# Include lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../../lib.Makefile

###############################################################################
# Build
###############################################################################
EKS_SRC_FILES = $(shell find eks/ -name '*.go')
EKS_VERSION_FLAGS=-X main.VERSION=$(GIT_VERSION) \
	-X main.BUILD_DATE=$(DATE) \
	-X main.GIT_TAG=$(GIT_DESCRIPTION) \
	-X main.GIT_COMMIT=$(GIT_COMMIT)

.PHONY: build
build: bin/eks-log-forwarder-startup-$(ARCH)

## build cloudwatch plugin initializer
bin/eks-log-forwarder-startup-$(ARCH): $(EKS_SRC_FILES)
	@echo "Building eks init-container executable"
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) CGO_ENABLED=0 go build -C eks -o $@ -v -ldflags "$(EKS_VERSION_FLAGS) -s -w"'

bin/eks-log-forwarder-startup.exe: $(EKS_SRC_FILES)
	@echo "Building eks init-container executable for windows"
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) CGO_ENABLED=0 GOARCH=amd64 GOOS=windows OS=windows go build -C eks -o $@ -v -ldflags "$(EKS_VERSION_FLAGS) -s -w"'

clean:
	rm -rf eks/bin Makefile.common*
	-docker image rm -f $$(docker images $(FLUENTD_IMAGE) -a -q)

###############################################################################
# Image
###############################################################################
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES)) 
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image:
	$(MAKE) $(addprefix build-image-,$(VALIDARCHES))

build-image-%:
ifeq ($(OS),Windows_NT)
	docker build --pull -t $(FLUENTD_IMAGE):latest-$* -f Dockerfile.windows .
else
	$(MAKE) build
	$(DOCKER_BUILD) -t $(FLUENTD_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
endif

###############################################################################
# CI/CD
###############################################################################
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

## test cloudwatch plugin initializer
ut: bin/eks-log-forwarder-startup-$(ARCH)
ifeq ($(ARCH),amd64)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test -C eks $(UNIT_TEST_FLAGS)'
endif

st:

## fluentd config tests
fv: image
ifeq ($(ARCH),amd64)
	cd test && IMAGETAG=latest-$(ARCH) IMAGE=tigera/$(FLUENTD_IMAGE) ./test.sh
endif

ci: test image

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
