include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/envoybinary

ENVOY_GATEWAY_IMAGE ?= tigera/envoy-gateway
BUILD_IMAGES ?= $(ENVOY_GATEWAY_IMAGE)

ENVOY_GATEWAY_VERSION=v1.1.2

DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = $(ENVOY_VERSION)
LATEST_IMAGE_TAG = $(BRANCH_NAME)

##############################################################################
# Include lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../../lib.Makefile

##############################################################################
# Build
##############################################################################
ENVOY_GATEWAY_DOWNLOADED=.envoy-gateway.downloaded

.PHONY: init-source
init-source:: $(ENVOY_GATEWAY_DOWNLOADED)
$(ENVOY_GATEWAY_DOWNLOADED):
	mkdir -p envoy-gateway
	curl -sfL https://github.com/envoyproxy/gateway/archive/refs/tags/$(ENVOY_GATEWAY_VERSION).tar.gz | tar xz --strip-components 1 -C envoy-gateway
	touch $@

##############################################################################
# Image
##############################################################################
ENVOY_GATEWAY_IMAGE_CREATED=.envoy-gateway.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

# intermediate multi-arch image holds our patched envoy -gateway.
.PHONY: image
image: $(ENVOY_GATEWAY_IMAGE)

ENVOY_ARCH=$(ARCH)
ifeq ($(ARCH),amd64)
ENVOY_ARCH=x64
endif

$(ENVOY_GATEWAY_IMAGE): $(ENVOY_GATEWAY_IMAGE_CREATED)
$(ENVOY_GATEWAY_IMAGE_CREATED): init-source
	make -C envoy-gateway build image PLATFORMS="linux_$(ARCH)" BINS="envoy-gateway" REGISTRY=tigera IMAGE_NAME=envoy-gateway TAG=$(LATEST_IMAGE_TAG)-$(ARCH) GOARCH=$(ARCH)
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=$(LATEST_IMAGE_TAG)
	touch $@

##############################################################################
# CI/CD
##############################################################################
.PHONY: cd
cd: image-all var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME
	$(MAKE) retag-build-images-with-registries push-images-to-registries IMAGETAG=$(LATEST_IMAGE_TAG) EXCLUDEARCH="$(EXCLUDEARCH)"

.PHONY: push-manifest
push-manifest:
	$(MAKE) push-manifests IMAGETAG=$(LATEST_IMAGE_TAG)
