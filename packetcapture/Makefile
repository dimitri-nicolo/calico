# Copyright 2019-2022 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/packetcapture

#############################################
# Env vars related to packaging and releasing
#############################################
PACKETCAPTURE_IMAGE ?=packetcapture
BUILD_IMAGES        ?=$(PACKETCAPTURE_IMAGE)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES = $(shell find pkg cmd -name '*.go') $(shell find ../libcalico-go/lib -name '*.go') $(shell find ../lma/pkg -name '*.go')

# Exclude deprecation warnings (SA1019), since failing on deprecation defeats the purpose
# of deprecating.
LINT_ARGS += --exclude SA1019

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/packetcapture-$(ARCH)

.PHONY: bin/packetcapture-$(ARCH)
bin/packetcapture-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go && \
				( ldd $@ 2>&1 | \
					grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# BUILD IMAGE
###############################################################################
# Build the docker image.
.PHONY: $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(PACKETCAPTURE_IMAGE) $(PACKETCAPTURE_IMAGE)-$(ARCH)
image: $(BUILD_IMAGES)

$(PACKETCAPTURE_IMAGE): $(PACKETCAPTURE_IMAGE)-$(ARCH)
$(PACKETCAPTURE_IMAGE)-$(ARCH): bin/packetcapture-$(ARCH)
	$(DOCKER_BUILD) -t $(PACKETCAPTURE_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

.PHONY: clean
clean:
	rm -rf bin \
		report/*.xml \
		release-notes-*
	-docker image rm -f $$(docker images $(PACKETCAPTURE_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
MOCKERY_FILE_PATHS= \
	pkg/cache/ClientCache \
	pkg/capture/FileCommands \
	pkg/capture/K8sCommands

GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################

.PHONY: ut
## Run only Unit Tests.
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download && $(GINKGO) -r pkg/*'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut

## Deploys images to registry
cd: image-all cd-common
