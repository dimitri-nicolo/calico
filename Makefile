# Copyright 2019 Tigera Inc. All rights reserved.

# The following is a generic Makefile, simply find & replace all mentions of YOUR_APP with 
# whatever the name of your component is.

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
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")
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
YOUR_APP_VERSION?=$(shell git describe --tags --dirty --always)
YOUR_APP_BUILD_DATE?=$(shell date -u +'%FT%T%z')
YOUR_APP_GIT_REVISION?=$(shell git rev-parse --short HEAD)
YOUR_APP_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(YOUR_APP_VERSION) \
	-X main.BUILD_DATE=$(YOUR_APP_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(YOUR_APP_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(YOUR_APP_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

#############################################
# Env vars related to building, packaging 
# and releasing
#############################################
BUILD_IMAGE?=tigera/$(YOUR_APP)
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=quay.io/$(BUILD_IMAGE)
PACKAGE_NAME?=github.com/tigera/$(YOUR_APP)
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
GO_BUILD_VER?=v0.21
CALICO_BUILD=calico/go-build:$(GO_BUILD_VER)

DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   mkdir -p .go-build-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -e GOARCH=$(ARCH) \
                              -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
                              -v ${PWD}:/$(PACKAGE_NAME):rw \
                              -v ${PWD}/.go-pkg-cache:/go/pkg:rw \
                              -v ${PWD}/.go-build-cache:/home/user/.cache/go-build:rw \
                              -w /$(PACKAGE_NAME) \
                              $(CALICO_BUILD)

##########################################################################################
# Display usage output
##########################################################################################
help:
	@echo "Tigera $(YOUR_APP) Makefile"
	@echo "Builds:"
	@echo
	@echo "  make all                   Build all the binary packages."
	@echo "  make YOUR_APP:             Build binary package for YOUR_APP."
	@echo "  make tigera/YOUR_APP       Build $(BUILD_IMAGE) docker image."
	@echo "  make image                 Build $(BUILD_IMAGE) docker image."
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

all: $(BUILD_IMAGE)

##########################################################################################
# BUILD
##########################################################################################

#############################################
# Golang Binary
#############################################

# Some will have dedicated targets to make it easier to type, for example
# "YOUR_APP" instead of "$(BINDIR)/YOUR_APP".
$(YOUR_APP): $(BINDIR)/$(YOUR_APP)

$(BINDIR)/$(YOUR_APP): $(BINDIR)/$(YOUR_APP)-amd64
	cd $(BINDIR) && (rm -f $(YOUR_APP); ln -s $(YOUR_APP)-$(ARCH) $(YOUR_APP))


$(BINDIR)/$(YOUR_APP)-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building $(YOUR_APP)...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	           go build -o $@ -v $(LDFLAGS) cmd/$(YOUR_APP)/*.go && \
               ( ldd $(BINDIR)/$(YOUR_APP)-$(ARCH) 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $(BINDIR)/$(YOUR_APP)-$(ARCH) was not statically linked"; false ) )'

#############################################
# Docker Image
#############################################

image: $(BUILD_IMAGE)
$(BUILD_IMAGE): $(BUILD_IMAGE)-$(ARCH)
$(BUILD_IMAGE)-$(ARCH): $(BINDIR)/$(YOUR_APP)-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp $(BINDIR)/$(YOUR_APP)-$(ARCH) docker-image/bin/
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) --file ./docker-image/Dockerfile-$(YOUR_APP).$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
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
		$(CALICO_BUILD) sh -c 'cd /build-dir/$(PACKAGE_NAME) && ginkgo -cover -r pkg/* internal/* $(GINKGO_ARGS)'

#############################################
# Run package level functional level tests
#############################################
.PHONY: fv
fv:
	echo "FV not implemented yet"

##########################################################################################
# CLEAN UP 
##########################################################################################
.PHONY: clean
clean: clean-bin clean-build-image

clean-build-image:
	docker rmi -f $(BUILD_IMAGE) > /dev/null 2>&1 || true

clean-bin:
	rm -rf $(BINDIR) \
			docker-image/bin

##########################################################################################
# CI/CD
##########################################################################################
.PHONY: ci cd

#############################################
# Run CI cycle - build, test, etc.
#############################################
ci: test 
	echo "CI not implemented yet"

#############################################
# Deploy images to registry
#############################################
cd:
	echo "CD not implemented yet"
