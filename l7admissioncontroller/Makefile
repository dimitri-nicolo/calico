include ../metadata.mk

PACKAGE_NAME = github.com/projectcalico/calico/l7admissioncontroller

###############################################################################

SRC_FILES=$(shell find . -name '*.go')

# Name of the images.
# e.g., <registry>/<name>:<tag>
L7ADMCTRL_IMAGE ?=l7-admission-controller
BUILD_IMAGES ?= $(L7ADMCTRL_IMAGE)

LDFLAGS = -X main.VERSION=$(GIT_VERSION)

##############################################################################
# Download and include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

CONTAINER_CREATED=.image.created-$(ARCH)
CONTAINER_FIPS_CREATED=.image.created-$(ARCH)-fips

FIPS ?= false

ifeq ($(FIPS),true)
CONTAINER_MARKER=$(CONTAINER_FIPS_CREATED)
VALIDARCHES=amd64
BINDIR=bin/$(ARCH)-fips
else
CONTAINER_MARKER=$(CONTAINER_CREATED)
BINDIR=bin
endif

# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	rm -rf bin report
	-docker image rm -f $$(docker images $(L7ADMCTRL_IMAGE) -a -q)

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
$(VALIDARCHES):
	$(MAKE) build ARCH=$@

build-all: $(VALIDARCHES)

.PHONY: build
build:
	$(MAKE) $(BINDIR)/l7admctrl-$(ARCH) ARCH=$(ARCH)

$(BINDIR)/l7admctrl-amd64: ARCH=amd64
$(BINDIR)/l7admctrl-arm64: ARCH=arm64
$(BINDIR)/l7admctrl-%: $(SRC_FILES)
	$(call build_cgo_binary, ./cmd/l7admctrl, $@)

###############################################################################
# Building the image
###############################################################################
.PHONY: image $(L7ADMCTRL_IMAGE)
image: $(L7ADMCTRL_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image FIPS=$(FIPS) ARCH=$*

$(L7ADMCTRL_IMAGE): $(CONTAINER_MARKER)
$(CONTAINER_CREATED): Dockerfile $(BINDIR)/l7admctrl-$(ARCH)
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(L7ADMCTRL_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

$(CONTAINER_FIPS_CREATED): Dockerfile $(BINDIR)/l7admctrl-$(ARCH)
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(L7ADMCTRL_IMAGE):latest-fips-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest-fips LATEST_IMAGE_TAG=latest-fips
	touch $@
###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev
ut:

###############################################################################
# CI
###############################################################################

.PHONY: ci
ci: static-checks image ut

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: image-all cd-common
