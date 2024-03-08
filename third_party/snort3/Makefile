# Copyright 2024 Tigera Inc. All rights reserved.
include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/snort3

#############################################
# Env vars related to packaging and releasing
#############################################
SNORT3_IMAGE ?= snort3
BUILD_IMAGES ?= $(SNORT3_IMAGE)

###############################################################################
# Include lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../../lib.Makefile

###############################################################################
# BUILD IMAGE
###############################################################################
SNORT3_IMAGE_CREATED=.snort3.created-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(SNORT3_IMAGE)

$(SNORT3_IMAGE): $(SNORT3_IMAGE_CREATED)
$(SNORT3_IMAGE_CREATED): Dockerfile snort3-community.rules
	$(DOCKER_BUILD) -t $(SNORT3_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -f $(SNORT3_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(SNORT3_IMAGE) -a -q)
