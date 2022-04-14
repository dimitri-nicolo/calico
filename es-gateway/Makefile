include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/es-gateway
LOCAL_CHECKS     = mod-download

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
SRC_FILES       = $(shell find pkg cmd -name '*.go')
ES_GATEWAY_IMAGE   ?=tigera/es-gateway

BUILD_VERSION         ?= $(shell git describe --tags --dirty --always 2>/dev/null)
BUILD_BUILD_DATE      ?= $(shell date -u +'%FT%T%z')
BUILD_GIT_DESCRIPTION ?= $(shell git describe --tags 2>/dev/null)
BUILD_GIT_REVISION    ?= $(shell git rev-parse --short HEAD)

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(BUILD_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(BUILD_BUILD_DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(BUILD_GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(BUILD_GIT_REVISION)

BUILD_LDFLAGS   = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: es-gateway

.PHONY: es-gateway bin/es-gateway bin/es-gateway-$(ARCH)
es-gateway: bin/es-gateway

bin/es-gateway: bin/es-gateway-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd bin && ln -sf es-gateway-$(ARCH) es-gateway'

bin/es-gateway-$(ARCH): $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) cmd/$*/*.go && \
				( ldd $@ 2>&1 | \
					grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# BUILD IMAGE
###############################################################################
# Build the docker image.
.PHONY: $(ES_GATEWAY_IMAGE) $(ES_GATEWAY_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(ES_GATEWAY_IMAGE)
image: $(ES_GATEWAY_IMAGE)
$(ES_GATEWAY_IMAGE): $(ES_GATEWAY_IMAGE)-$(ARCH)
$(ES_GATEWAY_IMAGE)-$(ARCH): bin/es-gateway-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/es-gateway-$(ARCH) docker-image/bin/
	docker build --pull -t $(ES_GATEWAY_IMAGE):latest-$(ARCH) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(ES_GATEWAY_IMAGE):latest-$(ARCH) $(ES_GATEWAY_IMAGE):latest
endif

.PHONY: clean
clean:
	rm -rf bin \
		docker-image/bin \
		report/*.xml \
		release-notes-* \
		Makefile.common*
	docker rmi -f $(ES_GATEWAY_IMAGE) > /dev/null 2>&1

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
