# Copyright 2019-20 Tigera Inc. All rights reserved.

PACKAGE_NAME   ?= github.com/tigera/voltron
GO_BUILD_VER   ?= v0.65
GIT_USE_SSH     = true
LOCAL_CHECKS    = mod-download

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_VOLTRON_PROJECT_ID)

#############################################
# Env vars related to packaging and releasing
#############################################
COMPONENTS            ?=guardian voltron
VOLTRON_IMAGE         ?=tigera/voltron
GUARDIAN_IMAGE        ?=tigera/guardian
BUILD_IMAGES          ?= $(addprefix tigera/, $(COMPONENTS))
ARCHES                ?=amd64
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# Mount Semaphore configuration files.
ifdef ST_MODE
EXTRA_DOCKER_ARGS = -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp:rw -v /home/runner/config:/home/runner/config:rw -v /home/runner/docker_auth.json:/home/runner/config/docker_auth.json:rw
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

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

include Makefile.common

##########################################################################################
# Define some constants
##########################################################################################
# Exclude deprecation warnings (SA1019), since failing on deprecation defeats the purpose
# of deprecating.
LINT_ARGS += --exclude SA1019

BRANCH_NAME ?= $(PIN_BRANCH)

# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

K8S_VERSION    = v1.10.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go -exec dirname {} \\; | sort -u")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort -u")
GO_FILES       = $(shell sh -c "find . -type f -name '*.go' -not -path './.go-pkg-cache/*' | sort -u")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

#############################################
# Env vars related to building
#############################################
BUILD_VERSION         ?= $(shell git describe --tags --dirty --always 2>/dev/null)
BUILD_BUILD_DATE      ?= $(shell date -u +'%FT%T%z')
BUILD_GIT_REVISION    ?= $(shell git rev-parse --short HEAD)
BUILD_GIT_DESCRIPTION ?= $(shell git describe --tags 2>/dev/null)
GIT_VERSION_LONG       = $(shell git describe --tags --dirty --always --long --abbrev=12)

# Flags for building the binaries.
#
# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(BUILD_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(BUILD_BUILD_DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(BUILD_GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(BUILD_GIT_REVISION) \

BUILD_LDFLAGS   = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

##########################################################################################
# BUILD
##########################################################################################
build: $(COMPONENTS)
$(COMPONENTS): %: $(BINDIR)/% ;

$(BINDIR)/%: $(BINDIR)/%-$(ARCH)
	rm -f $@; ln -s $*-$(ARCH) $@

$(BINDIR)/%-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building $* ...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	        go build -o $@ -v $(LDFLAGS) cmd/$*/*.go && \
               ( ldd $@ 2>&1 | \
	               grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
	             ( echo "Error: $@ was not statically linked"; false ) )'

#############################################
# Docker Image
#############################################

## Build all app docker images.
images: build $(BUILD_IMAGES)
tigera/%: tigera/%-$(ARCH) ;
tigera/%-$(ARCH): $(BINDIR)/%-$(ARCH)
	rm -rf docker-image/$*/bin
	mkdir -p docker-image/$*/bin
	cp $(BINDIR)/$*-$(ARCH) docker-image/$*/bin/
	docker build --pull -t tigera/$*:latest-$(ARCH) ./docker-image/$*
ifeq ($(ARCH),amd64)
	docker tag tigera/$*:latest-$(ARCH) tigera/$*:latest
endif

##########################################################################################
# TESTING
##########################################################################################

GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################

.PHONY: ut
## Run only Unit Tests.
ut: CMD = go mod download && $(GINKGO) -r pkg/* internal/*
ut:
ifdef LOCAL
	$(CMD)
else
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(CMD)'
endif

#############################################
# Run package level functional level tests
#############################################
.PHONY: fv
## Run only Package Functional Tests.
fv: CMD = go mod download && $(GINKGO) -r test/fv
fv:
ifdef LOCAL
	$(CMD)
else
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(CMD)'
endif

##########################################################################################
# CLEAN UP
##########################################################################################
.PHONY: clean
## Remove binary files.
clean: clean-build-image
	rm -rf $(BINDIR) docker-image/bin vendor
	find . -name "*.coverprofile" -type f -delete

clean-build-image:
	# Remove all variations e.g. tigera/voltron:latest + tigera/voltron:latest-amd64
	docker rmi -f $(BUILD_IMAGES) $(addsuffix :latest-$(ARCH), $(BUILD_IMAGES)) > /dev/null 2>&1 || true

###############################################################################
# Static checks
###############################################################################

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed := $(shell ./install-git-hooks.sh)

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

API_REPO = github.com/tigera/api
LMA_REPO=github.com/tigera/lma
LMA_BRANCH=$(PIN_BRANCH)

update-lma-pin:
	$(call update_pin,$(LMA_REPO),$(LMA_REPO),$(LMA_BRANCH))

update-pins: guard-ssh-forwarding-bug update-lma-pin update-calico-pin

##########################################################################################
# CI/CD
##########################################################################################
.PHONY: ci cd

#############################################
# Run CI cycle - build, test, etc.
#############################################
## Run all CI steps for build and test, likely other targets.
ci: clean static-checks ut fv images

#############################################
# Deploy images to registry
#############################################
## Run all CD steps, normally pushing images out to registries.
cd: images cd-common

##########################################################################################
# LOCAL RUN
##########################################################################################

run-voltron:
	VOLTRON_CERT_PATH=test go run cmd/voltron/main.go

run-guardian:
	GUARDIAN_VOLTRON_URL=127.0.0.1:5555 go run cmd/guardian/main.go

##########################################################################################
# MOCKING
##########################################################################################

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS=\
	pkg/tunnel/Dialer
