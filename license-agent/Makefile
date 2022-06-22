PACKAGE_NAME    ?= github.com/tigera/license-agent
GO_BUILD_VER    ?= v0.65
GIT_USE_SSH      = true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private
LOCAL_CHECKS     = mod-download

build: license-agent

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_LICENSE_AGENT_PROJECT_ID)

LICENSE_AGENT_IMAGE        ?=tigera/license-agent
BUILD_IMAGES               ?=$(LICENSE_AGENT_IMAGE)
DEV_REGISTRIES             ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES         ?=quay.io
RELEASE_BRANCH_PREFIX      ?=release-calient
DEV_TAG_SUFFIX             ?=calient-0.dev

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*
# Allow local libcalico-go to be mapped into the build container.
ifdef CALICO_PATH
EXTRA_DOCKER_ARGS += -v $(CALICO_PATH):/go/src/github.com/projectcalico/calico/:ro
endif
# SSH_AUTH_DIR doesn't work with MacOS but we can optionally volume mount keys
ifdef SSH_AUTH_DIR
EXTRA_DOCKER_ARGS += --tmpfs /home/user -v $(SSH_AUTH_DIR):/home/user/.ssh:ro
endif

include Makefile.common

K8S_VERSION    = v1.11.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

LICENSE_AGENT_VERSION?=$(shell git describe --tags --dirty --always)
LICENSE_AGENT_BUILD_DATE?=$(shell date -u +'%FT%T%z')
LICENSE_AGENT_GIT_DESCRIPTION?=$(shell git describe --tags)
LICENSE_AGENT_GIT_REVISION?=$(shell git rev-parse --short HEAD)

VERSION_FLAGS=-X main.VERSION=$(LICENSE_AGENT_VERSION) \
				-X main.BUILD_DATE=$(LICENSE_AGENT_BUILD_DATE) \
				-X main.GIT_DESCRIPTION=$(LICENSE_AGENT_GIT_DESCRIPTION) \
				-X main.GIT_REVISION=$(LICENSE_AGENT_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "license-agent" instead of "$(BINDIR)/license-agent".
license-agent: $(BINDIR)/license-agent

$(BINDIR)/license-agent: $(BINDIR)/license-agent-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd $(BINDIR) && ln -s -T license-agent-$(ARCH) license-agent'

$(BINDIR)/license-agent-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building license-agent...
	# configure git to use ssh instead of https so that go mod can pull private libraries.
	# note this will require the user have their SSH agent running and configured with valid private keys
	# but the Makefile logic here will load the local SSH agent into the container automatically.
	mkdir -p .go-build-cache && \
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url.ssh://git@github.com.insteadOf https://github.com && \
			go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/server" && \
				( ldd $(BINDIR)/license-agent-$(ARCH) 2>&1 | \
				grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $(BINDIR)/license-agent-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(LICENSE_AGENT_IMAGE) $(LICENSE_AGENT_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(LICENSE_AGENT_IMAGE)
$(LICENSE_AGENT_IMAGE): $(LICENSE_AGENT_IMAGE)-$(ARCH)
$(LICENSE_AGENT_IMAGE)-$(ARCH): $(BINDIR)/license-agent-$(ARCH)
	docker build --pull -t $(LICENSE_AGENT_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(LICENSE_AGENT_IMAGE):latest-$(ARCH) $(LICENSE_AGENT_IMAGE):latest
endif

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

.PHONY: clean
clean: clean-bin clean-build-image
clean-build-image:
	docker rmi -f $(LICENSE_AGENT_IMAGE) > /dev/null 2>&1 || true

clean-bin:
	rm -rf $(BINDIR) bin


.PHONY: signpost
signpost:
	@echo "------------------------------------------------------------------------------"

###############################################################################
# Static checks
###############################################################################

###############################################################################
# See .golangci.yml for golangci-lint config
LINT_ARGS +=

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean image-all static-checks ut

## Deploys images to registry
cd: image-all cd-common

###############################################################################
# Update pins
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

API_REPO=github.com/tigera/api

## Update dependency pins
update-pins: guard-ssh-forwarding-bug update-calico-pin

###############################################################################
# Utilities
###############################################################################
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')
