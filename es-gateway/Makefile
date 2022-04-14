PACKAGE_NAME    ?= github.com/tigera/es-gateway
GO_BUILD_VER    ?= v0.65
GIT_USE_SSH      = true

LOCAL_CHECKS     = mod-download

GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_ES_GATEWAY_PROJECT_ID)

ES_GATEWAY_IMAGE   ?=tigera/es-gateway
BUILD_IMAGES       ?=$(ES_GATEWAY_IMAGE)
DEV_REGISTRIES     ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES ?=quay.io

RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev


EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

###############################################################################
# Env vars related to building
###############################################################################
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
		sh -c 'cd bin && ln -s -T es-gateway-$(ARCH) es-gateway'

bin/es-gateway-$(ARCH): $(GO_FILES)
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
	rm -rf .go-pkg-cache \
		bin \
		docker-image/bin \
		report/*.xml \
		release-notes-* \
		vendor \
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
# Updating pins
###############################################################################	
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))

## Update dependency pins
update-pins: guard-ssh-forwarding-bug update-calico-pin

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
