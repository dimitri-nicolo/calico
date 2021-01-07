# Set repo config.
BUILD_IMAGE?=tigera/fluentd

ifeq ($(OS),Windows_NT)
DOCKERFILE?=Dockerfile.windows
else
DOCKERFILE?=Dockerfile
endif

# Override shell if we're on Windows
# https://stackoverflow.com/a/63840549
ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command
endif

# We append a tag to identify the arch the container image targets. For example, "$(VERSION)-windows-1903"
#
# We support these platforms:
# - Linux/amd64
# - Windows 10 1809 amd64
# - Windows 10 1903 amd64
# - Windows 10 1909 amd64
# - Windows 10 2004 amd64
ifeq ($(OS),Windows_NT)
$(eval WINDOWS_VERSION := $(shell (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").ReleaseId))
ARCH_TAG=-windows-$(WINDOWS_VERSION)
else
ARCH_TAG=-linux-amd64
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

## fluentd config tests
test: image eks-log-forwarder-startup
	cd $(SRC_DIR)/test && IMAGETAG=$(IMAGETAG)$(ARCH_TAG) ./test.sh && cd $(SRC_DIR)
	$(MAKE) -C eks/ ut

clean: clean-eks-log-forwarder-startup

## build fluentd image, tagged to IMAGETAG
ci: eks-log-forwarder-startup test image

## push fluentd image to GCR_REPO
cd: eks-log-forwarder-startup image
	$(MAKE) push VERSION=$(IMAGETAG)$(ARCH_TAG)
	$(MAKE) push VERSION=$(shell git describe --tags --dirty --always --long --abbrev=12)$(ARCH_TAG)
