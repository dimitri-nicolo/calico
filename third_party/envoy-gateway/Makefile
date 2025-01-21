include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/envoy-gateway

ENVOY_GATEWAY_IMAGE ?= tigera/envoy-gateway
BUILD_IMAGES ?= $(ENVOY_GATEWAY_IMAGE)

ENVOY_GATEWAY_VERSION=v1.1.2

DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = $(ENVOY_GATEWAY_VERSION)
LATEST_IMAGE_TAG = latest

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

.PHONY: build
build: bin/envoy-gateway-$(ARCH)

LD_FLAGS = \
	-X github.com/envoyproxy/gateway/internal/cmd/version.envoyGatewayVersion=$(ENVOY_GATEWAY_VERSION) \
	-X github.com/envoyproxy/gateway/internal/cmd/version.gitCommitID=$(GIT_COMMIT) \
        -s -w

bin/envoy-gateway-$(ARCH): init-source
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			CGO_ENABLED=0 go build -C envoy-gateway -buildvcs=false -o ../bin/envoy-gateway-$(ARCH) -ldflags="$(LD_FLAGS)" github.com/envoyproxy/gateway/cmd/envoy-gateway'

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

$(ENVOY_GATEWAY_IMAGE): $(ENVOY_GATEWAY_IMAGE_CREATED)
$(ENVOY_GATEWAY_IMAGE_CREATED): build
	$(DOCKER_BUILD) -t $(ENVOY_GATEWAY_IMAGE):latest-$(ARCH) -f Dockerfile .
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
