# Copyright 2023 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME?=github.com/projectcalico/calico/webhooks-processor

#############################################
# Env vars related to packaging and releasing
#############################################
WEBHOOKS_IMAGE?=webhooks-processor
BUILD_IMAGES?=$(WEBHOOKS_IMAGE)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES = $(shell find pkg cmd -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \

# We use -X to insert the version information into the placeholder variables
# in the version package.
LDFLAGS = -X $(PACKAGE_NAME)/pkg/config/version.BuildVersion=$(GIT_VERSION) \
			-X $(PACKAGE_NAME)/pkg/config/version.BuildDate=$(DATE) \
			-X $(PACKAGE_NAME)/pkg/config/version.GitDescription=$(GIT_DESCRIPTION) \
			-X $(PACKAGE_NAME)/pkg/config/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/webhooks-processor-$(ARCH)

.PHONY: bin/webhooks-processor-$(ARCH)
bin/webhooks-processor-$(ARCH): $(SRC_FILES)
	$(call build_static_cgo_boring_binary, cmd/$*/*.go, $@)

###############################################################################
# BUILD IMAGE
###############################################################################
WEBHOOKS_PROCESSOR_CONTAINER_CREATED=.webhooks-processor.created-$(ARCH)

# Build the docker image.
.PHONY: $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(WEBHOOKS_IMAGE)

$(WEBHOOKS_IMAGE): $(WEBHOOKS_PROCESSOR_CONTAINER_CREATED)
$(WEBHOOKS_PROCESSOR_CONTAINER_CREATED): docker-image/Dockerfile.$(ARCH) bin/webhooks-processor-$(ARCH)
	$(DOCKER_BUILD) -t $(WEBHOOKS_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@
	
.PHONY: ut
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./pkg/... -cover -count 1'

.PHONY: clean
clean:
	-rm -f ./bin/*
	-rm -f $(WEBHOOKS_PROCESSOR_CONTAINER_CREATED)
	-docker rmi -f $$(docker images -aq $(WEBHOOKS_IMAGE))
