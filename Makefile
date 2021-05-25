GO_BUILD_VER?= v0.51

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_FLUENTD_DOCKER_PROJECT_ID)

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

ifeq ($(OS),Windows_NT)
DOCKERFILE?=Dockerfile.windows
BUILD_IMAGE?=tigera/fluentd-windows
else
DOCKERFILE?=Dockerfile
BUILD_IMAGE?=tigera/fluentd
endif

# Override shell if we're on Windows
# https://stackoverflow.com/a/63840549
ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command
endif

# For Windows we append to the image tag to identify the Windows 10 version.
# For example, "v3.5.0-calient-0.dev-26-gbaba2f0b96a4-windows-1903"
#
# We support these platforms:
# - Windows 10 1809 amd64
# - Windows 10 2004 amd64
# - Windows 10 20H2 amd64
#
# For Linux, we leave the image tag alone.
ifeq ($(OS),Windows_NT)
# Get the Windows build number.
$(eval WINDOWS_BUILD_VERSION := $(shell (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuild))

# Get Windows version based on build number.
ifeq ($(WINDOWS_BUILD_VERSION),17763)
WINDOWS_VERSION := 1809
else ifeq ($(WINDOWS_BUILD_VERSION),19041)
WINDOWS_VERSION := 2004
else ifeq ($(WINDOWS_BUILD_VERSION),19042)
WINDOWS_VERSION := 20H2
else
$(error Unknown WINDOWS_BUILD_VERSION)
endif

ARCH_TAG=-windows-$(WINDOWS_VERSION)
else
ARCH_TAG=
endif

SRC_DIR?=$(PWD)
# Overwrite configuration, e.g. GCR_REPO:=gcr.io/tigera-dev/experimental/gaurav

default: ci

-include makefile.tigera

# Setup custom targets
.PHONY: ci cd
## build cloudwatch plugin initializer
eks-log-forwarder-startup:
	$(MAKE) -C eks/ eks-log-forwarder-startup

## clean slate cloudwatch plugin initializer
clean-eks-log-forwarder-startup:
	$(MAKE) -C eks/ clean

## test cloudwatch plugin initializer
test-eks-log-forwarder-startup: eks-log-forwarder-startup
	$(MAKE) -C eks/ ut

ut:

## fluentd config tests
fv: image eks-log-forwarder-startup
	cd $(SRC_DIR)/test && IMAGETAG=$(IMAGETAG)$(ARCH_TAG) ./test.sh && cd $(SRC_DIR)
	$(MAKE) -C eks/ ut

st:

clean: clean-eks-log-forwarder-startup
	rm -rf Makefile.common*

## build fluentd image, tagged to IMAGETAG

ci: eks-log-forwarder-startup test image

## push fluentd image to GCR_REPO.
#  Note: this is called from both Linux and Windows so ARCH_TAG is required.
cd: eks-log-forwarder-startup image
	$(MAKE) push VERSION=$(IMAGETAG)$(ARCH_TAG)
	$(MAKE) push VERSION=$(shell git describe --tags --dirty --always --long --abbrev=12)$(ARCH_TAG)

## push fluentd-windows image manifest to GCR_REPO
#  Note: this should only be invoked after 'make cd' is run for all supported
#  Windows versions
cd-windows-manifest:
	$(MAKE) push-windows-manifest VERSION=$(IMAGETAG)
	$(MAKE) push-windows-manifest VERSION=$(shell git describe --tags --dirty --always --long --abbrev=12)
