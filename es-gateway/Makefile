include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/es-gateway

RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

###############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES        = $(shell find pkg cmd -name '*.go') $(shell find ../libcalico-go/lib/logutils -name '*.go')
ES_GATEWAY_IMAGE ?=es-gateway
BUILD_IMAGES     ?=$(ES_GATEWAY_IMAGE)

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
build: bin/es-gateway-$(ARCH)

.PHONY: bin/es-gateway-$(ARCH)
bin/es-gateway-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go && \
				( ldd $@ 2>&1 | \
					grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# Building the image
###############################################################################
.PHONY: image $(ES_GATEWAY_IMAGE) $(ES_GATEWAY_IMAGE)-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(ES_GATEWAY_IMAGE)
$(ES_GATEWAY_IMAGE): $(ES_GATEWAY_IMAGE)-$(ARCH)
$(ES_GATEWAY_IMAGE)-$(ARCH): bin/es-gateway-$(ARCH)
	$(DOCKER_BUILD) -t $(ES_GATEWAY_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

.PHONY: clean
clean:
	rm -rf bin
	-docker image rm -f $$(docker images $(ES_GATEWAY_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################

.PHONY: ut
## Run only Unit Tests.
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download && $(GINKGO) pkg/*'

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
