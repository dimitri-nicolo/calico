# Copyright 2019 Tigera Inc. All rights reserved.

APP_NAME=voltron
COMPONENTS=guardian voltron

##########################################################################################
# Define some constants
##########################################################################################

# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst docker-image/Dockerfile.%,%,$(wildcard docker-image/Dockerfile.*))

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)
BUILDOS ?= $(shell uname -s | tr A-Z a-z)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
        BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
        BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
    override ARCH=amd64
endif


# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

K8S_VERSION    = v1.10.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
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
# Env vars for 
#############################################
BUILD_VERSION?=$(shell git describe --tags --dirty --always)
BUILD_BUILD_DATE?=$(shell date -u +'%FT%T%z')
BUILD_GIT_REVISION?=$(shell git rev-parse --short HEAD)
BUILD_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(BUILD_VERSION) \
	-X main.BUILD_DATE=$(BUILD_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(BUILD_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(BUILD_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

#############################################
# Env vars related to building, packaging 
# and releasing
#############################################
BUILD_IMAGES?=$(addprefix tigera/, $(COMPONENTS))
PACKAGE_NAME?=github.com/tigera/$(APP_NAME)
RELEASE_BUILD?=""

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
GO_BUILD_VER?=go.1.12
CALICO_BUILD= aliceinwonderland89/go-build:${GO_BUILD_VER}

DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   mkdir -p .go-build-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -e GOCACHE=/$(PACKAGE_NAME)/.go-build-cache \
                              -e GOARCH=$(ARCH) \
                              -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
                              -v ${PWD}:/$(PACKAGE_NAME):rw \
                              -v ${PWD}/.go-pkg-cache:/go/pkg:rw \
                              -w /$(PACKAGE_NAME) \
                              $(CALICO_BUILD)

##########################################################################################
# Display usage output
##########################################################################################
help:
	@echo "Tigera $(APP_NAME) Makefile"
	@echo "Builds:"
	@echo
	@echo "  make all                   Build all the binary packages."
	@echo "  make <component>:          Build <component> binary."
	@echo "  make tigera/<component>    Build the components docker image."
	@echo "  make images                Build all app docker images."
	@echo
	@echo "CI & CD:"
	@echo
	@echo "  make ci                    Run all CI steps for build and test, likely other targets."
	@echo "  make cd                    Run all CD steps, normally pushing images out to registries."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make test                  Run all Tests."
	@echo "  make ut                    Run only Unit Tests."
	@echo "  make fv                    Run only Package Functional Tests."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make clean                 Remove binary files."

# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

all: images

##########################################################################################
# BUILD
##########################################################################################

#############################################
# Golang Binary
#############################################

$(COMPONENTS): %: $(BINDIR)/% ;

.PRECIOUS:$(BINDIR)/%-$(ARCH)

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
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	           go build -o $@ -v $(LDFLAGS) cmd/$*/*.go && \
               ( ldd $@ 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $@ was not statically linked"; false ) )'

#############################################
# Docker Image
#############################################

images: $(BUILD_IMAGES)
tigera/%: tigera/%-$(ARCH) ;
tigera/%-$(ARCH): $(BINDIR)/%-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp $(BINDIR)/$*-$(ARCH) docker-image/bin/
	mkdir -p docker-image/templates
	cp manifests/guardian.yaml docker-image/templates/
	docker build --pull -t tigera/$*:latest-$(ARCH) --file ./docker-image/Dockerfile.$*.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag tigera/$*:latest-$(ARCH) tigera/$*:latest
endif

##########################################################################################
# TESTING 
##########################################################################################
test: ut fv

#############################################
# Run unit level tests
#############################################
.PHONY: ut
ut:
	docker run --rm -v $(CURDIR):/build-dir/$(PACKAGE_NAME):rw \
		-e LOCAL_USER_ID=$(MY_UID) \
		$(CALICO_BUILD) sh -c 'cd /build-dir/$(PACKAGE_NAME) && go mod download && ginkgo -cover -r pkg/* internal/* $(GINKGO_ARGS)'

#############################################
# Run package level functional level tests
#############################################
.PHONY: fv
fv:
	docker run --rm -v $(CURDIR):/build-dir/$(PACKAGE_NAME):rw \
		-e LOCAL_USER_ID=$(MY_UID) \
		$(CALICO_BUILD) sh -c 'cd /build-dir/$(PACKAGE_NAME) && go mod download && ginkgo -cover -r test/fv $(GINKGO_ARGS)'

##########################################################################################
# CLEAN UP 
##########################################################################################
.PHONY: clean
clean: clean-bin clean-build-image clean-docker-image-templates

clean-build-image:
	# Remove all variations e.g. tigera/voltron:latest + tigera/voltron:latest-amd64
	docker rmi -f $(BUILD_IMAGES) $(addsuffix :latest-$(ARCH), $(BUILD_IMAGES)) > /dev/null 2>&1 || true

clean-bin:
	rm -rf $(BINDIR) \
			docker-image/bin

clean-docker-image-templates: 
	rm -rf docker-image/templates
	
###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
static-checks:
	docker run --rm -v $(CURDIR):/build-dir/$(PACKAGE_NAME):rw \
		-e LOCAL_USER_ID=$(MY_UID) \
		$(CALICO_BUILD) sh -c 'cd /build-dir/$(PACKAGE_NAME); ./static-checks.sh'

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks.sh)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks.sh

##########################################################################################
# CI/CD
##########################################################################################
.PHONY: ci cd

#############################################
# Run CI cycle - build, test, etc.
#############################################
ci: clean test

#############################################
# Deploy images to registry
#############################################
cd: clean images deploy
deploy: $(addprefix deploy-, $(BUILD_IMAGES))
# FIXME: This doesn't work (cannot have % with / in it) 
# https://stackoverflow.com/questions/21182990/makefile-is-it-possible-to-have-stem-with-slash
deploy-%:
	docker tag $*:latest gcr.io/tigera-dev/cnx/$*:latest
	gcloud docker -- push gcr.io/tigera-dev/cnx/$*:latest
