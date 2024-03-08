# Copyright 2024 Tigera Inc. All rights reserved.

include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/snort2

#############################################
# Env vars related to packaging and releasing
#############################################
SNORT2_IMAGE ?= snort2
BUILD_IMAGES ?= $(SNORT2_IMAGE)

SNORT_VERSION ?= 2.9.20

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
	$(DOCKER_BUILD) --build-arg SNORT_VERSION=$(SNORT_VERSION) -t $(SNORT2_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -f $(SNORT2_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(SNORT2_IMAGE) -a -q)
