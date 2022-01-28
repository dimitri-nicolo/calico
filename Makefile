# Copyright 2019-20 Tigera Inc. All rights reserved.

PACKAGE_NAME    ?= github.com/tigera/packetcapture-api
GO_BUILD_VER    ?= v0.65
GIT_USE_SSH      = true
LOCAL_CHECKS     = mod-download

GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_PACKETCAPTURE_API_PROJECT_ID)

#############################################
# Env vars related to packaging and releasing
#############################################
PACKETCAPTURE_API_IMAGE   ?=tigera/packetcapture-api
BUILD_IMAGES       ?=$(PACKETCAPTURE_API_IMAGE)
ARCHES             ?=amd64
DEV_REGISTRIES     ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES ?=quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

# Mount Semaphore configuration files.
ifdef ST_MODE
EXTRA_DOCKER_ARGS = -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp:rw -v /home/runner/config:/home/runner/config:rw -v /home/runner/docker_auth.json:/home/runner/config/docker_auth.json:rw
endif

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
# Exclude deprecation warnings (SA1019), since failing on deprecation defeats the purpose
# of deprecating.
LINT_ARGS += --exclude SA1019

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
build: clean packetcapture-api

.PHONY: packetcapture-api bin/packetcapture-api bin/packetcapture-api-$(ARCH)
packetcapture-api: bin/packetcapture-api

bin/packetcapture-api: bin/packetcapture-api-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd bin && ln -s -T packetcapture-api-$(ARCH) packetcapture-api'

bin/packetcapture-api-$(ARCH): $(GO_FILES)
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
.PHONY: $(PACKETCAPTURE_API_IMAGE) $(PACKETCAPTURE_API_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(PACKETCAPTURE_API_IMAGE)
image: $(PACKETCAPTURE_API_IMAGE)
$(PACKETCAPTURE_API_IMAGE): $(PACKETCAPTURE_API_IMAGE)-$(ARCH)
$(PACKETCAPTURE_API_IMAGE)-$(ARCH): bin/packetcapture-api-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/packetcapture-api-$(ARCH) docker-image/bin/
	docker build --pull -t $(PACKETCAPTURE_API_IMAGE):latest-$(ARCH) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(PACKETCAPTURE_API_IMAGE):latest-$(ARCH) $(PACKETCAPTURE_API_IMAGE):latest
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
	docker rmi -f $(PACKETCAPTURE_API_IMAGE) > /dev/null 2>&1

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
# Updating pins
###############################################################################
update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

## Update dependency pins

LMA_REPO=github.com/tigera/lma
LMA_BRANCH=$(PIN_BRANCH)

replace-lma-pin:
	$(call update_pin,$(LMA_REPO),$(LMA_REPO),$(LMA_BRANCH))

update-pins: guard-ssh-forwarding-bug replace-lma-pin update-calico-pin

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
