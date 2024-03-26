# Copyright 2024 Tigera Inc. All rights reserved.

include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/snort2

#############################################
# Env vars related to packaging and releasing
#############################################
SNORT2_IMAGE ?= snort
BUILD_IMAGES ?= $(SNORT2_IMAGE)
# Snort 2 rpm package is x86_64 only from the vendor.
ARCHES = amd64

SNORT2_VERSION ?= 2.9.20

# required for ci/cd
DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = v$(SNORT2_VERSION)
LATEST_IMAGE_TAG = $(BRANCH_NAME)

###############################################################################
# Include lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../../lib.Makefile

###############################################################################
# BUILD IMAGE
###############################################################################
SNORT2_IMAGE_CREATED=.snort2.created-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(SNORT2_IMAGE)

$(SNORT2_IMAGE): $(SNORT2_IMAGE_CREATED)
$(SNORT2_IMAGE_CREATED): Dockerfile community.rules
	$(DOCKER_BUILD) --build-arg SNORT2_VERSION=$(SNORT2_VERSION) -t $(SNORT2_IMAGE):$(LATEST_IMAGE_TAG)-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=$(LATEST_IMAGE_TAG)
	touch $@

.PHONY: clean
clean:
	rm -f $(SNORT2_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(SNORT2_IMAGE) -a -q)

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd
cd: image-all var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME
	$(MAKE) retag-build-images-with-registries push-images-to-registries IMAGETAG=$(LATEST_IMAGE_TAG) EXCLUDEARCH="$(EXCLUDEARCH)"

.PHONY: push-manifest
push-manifest:
	$(MAKE) push-manifests IMAGETAG=$(LATEST_IMAGE_TAG)
