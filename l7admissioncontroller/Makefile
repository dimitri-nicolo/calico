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

IMAGE_CREATED=.image.created-$(ARCH)

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
.PHONY: build
build: bin/l7admctrl-$(ARCH)
	$(MAKE) bin/l7admctrl-$(ARCH) ARCH=$(ARCH)

bin/l7admctrl-%: $(SRC_FILES)
	$(call build_binary, cmd/l7admctrl/*.go, $@)

###############################################################################
# Building the image
###############################################################################
.PHONY: image $(L7ADMCTRL_IMAGE)
image: $(L7ADMCTRL_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(L7ADMCTRL_IMAGE): $(IMAGE_CREATED)
$(IMAGE_CREATED): Dockerfile bin/l7admctrl-$(ARCH)
	$(DOCKER_BUILD) -t $(L7ADMCTRL_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
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
