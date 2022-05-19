include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/prometheus-service

##############################################################################
# Constants
##############################################################################
PROMETHEUS_SERVICE_IMAGE  ?=prometheus-service
BUILD_IMAGES          		?=$(PROMETHEUS_SERVICE_IMAGE)

PROMETHEUS_SERVICE_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
PROMETHEUS_SERVICE_BUILD_DATE?=$(shell date -u +'%FT%T%z')
PROMETHEUS_SERVICE_GIT_COMMIT?=$(shell git rev-parse --short HEAD)
PROMETHEUS_SERVICE_GIT_TAG?=$(shell git describe --tags)

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/handler.VERSION=$(PROMETHEUS_SERVICE_VERSION) \
	-X $(PACKAGE_NAME)/pkg/handler.BUILD_DATE=$(PROMETHEUS_SERVICE_BUILD_DATE) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_TAG=$(PROMETHEUS_SERVICE_GIT_TAG) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_COMMIT=$(PROMETHEUS_SERVICE_GIT_COMMIT) \
	-X main.VERSION=$(PROMETHEUS_SERVICE_VERSION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Create an extended go-build image with docker binary installed for use with st target
TOOLING_IMAGE?=calico/go-build-with-docker
TOOLING_IMAGE_VERSION?=v0.24
TOOLING_IMAGE_CREATED=.go-build-with-docker.created

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Build
###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "prometheus-service" instead of "bin/prometheus-service".

BUILD_DIR?=build
TOP_SRC_DIRS=pkg
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
SRC_FILES:= $(shell find prometheus-service -type f -name '*.go' -print)


build: prometheus-service
prometheus-service: bin/prometheus-service

bin/prometheus-service: bin/prometheus-service-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd bin && ln -s -T prometheus-service-$(ARCH) prometheus-service'

bin/prometheus-service-$(ARCH): $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/server" && \
				( ldd bin/prometheus-service-$(ARCH) 2>&1 | \
	                grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: bin/prometheus-service-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(BUILD_IMAGES) $(addsuffix -$(ARCH),$(BUILD_IMAGES))

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(PROMETHEUS_SERVICE_IMAGE)
$(PROMETHEUS_SERVICE_IMAGE): $(PROMETHEUS_SERVICE_IMAGE)-$(ARCH)
$(PROMETHEUS_SERVICE_IMAGE)-$(ARCH): bin/prometheus-service-$(ARCH)
	docker build --pull -t $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) $(PROMETHEUS_SERVICE_IMAGE):latest
endif

.PHONY: clean
clean:
	docker rmi -f $(PROMETHEUS_SERVICE_IMAGE):latest > /dev/null 2>&1
	docker rmi -f $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) > /dev/null 2>&1
	rm -rf bin .go-pkg-cache Makefile.common*

##########################################################################
# Testing
##########################################################################
report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url.ssh://git@github.com.insteadOf https://github.com && \
			go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv: image report-dir
	$(MAKE) fv-no-setup

## Developer friendly target to only run fvs and skip other
## setup steps.
.PHONY: fv-no-setup
fv-no-setup:
	PACKAGE_ROOT=$(CURDIR) \
			     GO_BUILD_IMAGE=$(CALICO_BUILD) \
		       PACKAGE_NAME=$(PACKAGE_NAME) \
		       GINKGO_ARGS='$(GINKGO_ARGS)' \
		       GOMOD_CACHE=$(GOMOD_CACHE) \
		       ./fv/run_test.sh

###############################################################################
# Static checks
###############################################################################
# See .golangci.yml for golangci-lint config
# SA1019 are deprecation checks, we don't want to fail on those because it means a library update that deprecates something
# requires immediate removing of the deprecated functions.
LINT_ARGS += --exclude SA1019

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
ci: clean image-all static-checks ut fv

## Deploys images to registry
cd: image-all cd-common

###############################################################################
# Update pins
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
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

LMA_BRANCH?=$(PIN_BRANCH)
LMA_REPO?=github.com/tigera/lma

update-lma-pin:
	$(call update_pin,$(LMA_REPO),$(LMA_REPO),$(LMA_BRANCH))

update-pins: guard-ssh-forwarding-bug update-calico-pin update-lma-pin

###############################################################################
# Utils
###############################################################################
# this is not a linked target, available for convenience.
.PHONY: tidy
## 'tidy' go modules.
tidy:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod tidy'
