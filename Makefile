# Copyright 2019 Tigera Inc. All rights reserved.

APP_NAME   = voltron
COMPONENTS = guardian voltron

##########################################################################################
# Define some constants
##########################################################################################

# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES = amd64

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
# Env vars for 
#############################################
BUILD_VERSION         ?= $(shell git describe --tags --dirty --always 2>/dev/null)
BUILD_BUILD_DATE      ?= $(shell date -u +'%FT%T%z')
BUILD_GIT_REVISION    ?= $(shell git rev-parse --short HEAD)
BUILD_GIT_DESCRIPTION ?= $(shell git describe --tags 2>/dev/null)

VERSION_FLAGS   = -X main.VERSION=$(BUILD_VERSION) \
                  -X main.BUILD_DATE=$(BUILD_BUILD_DATE) \
                  -X main.GIT_DESCRIPTION=$(BUILD_GIT_DESCRIPTION) \
                  -X main.GIT_REVISION=$(BUILD_GIT_REVISION)
BUILD_LDFLAGS   = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

#############################################
# Env vars related to building, packaging 
# and releasing
#############################################
PUSH_REPO     ?= gcr.io/unique-caldron-775/cnx/
BUILD_IMAGES  ?= $(addprefix tigera/, $(COMPONENTS))
PACKAGE_NAME  ?= github.com/tigera/$(APP_NAME)
RELEASE_BUILD ?= ""

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID := $(shell id -u)
MY_GID := $(shell id -g)

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
GO_BUILD_VER ?= go.1.12
CALICO_BUILD  = aliceinwonderland89/go-build:${GO_BUILD_VER}

DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -e LOCAL_GROUP_ID=$(MY_GID) \
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

.PRECIOUS: $(BINDIR)/%-$(ARCH)

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

images: $(BUILD_IMAGES) manifests
tigera/%: tigera/%-$(ARCH) ;
tigera/%-$(ARCH): $(BINDIR)/%-$(ARCH)
	rm -rf docker-image/$*/bin
	mkdir -p docker-image/$*/bin
	cp $(BINDIR)/$*-$(ARCH) docker-image/$*/bin/
	docker build --pull -t tigera/$*:latest-$(ARCH) ./docker-image/$*
ifeq ($(ARCH),amd64)
	docker tag tigera/$*:latest-$(ARCH) tigera/$*:latest
endif

.PHONY: manifests
manifests: manifests/voltron.yaml

manifests/voltron.yaml: manifests/voltron.yaml.tmpl
	scripts/certs/clean-self-signed.sh scripts/certs
	scripts/certs/self-signed.sh scripts/certs
# Handle differences in base64 between OS 
ifeq ($(shell uname -s),Darwin)
	CERT64=`base64 scripts/certs/voltron.crt` && \
		KEY64=`base64 scripts/certs/voltron.key` && \
		cat $< | sed "s;VOLTRON_CRT_BASE64;$$CERT64;" | sed "s;VOLTRON_KEY_BASE64;$$KEY64;" > $@
else
	CERT64=`base64 -w 0 scripts/certs/voltron.crt` && \
		KEY64=`base64 -w 0 scripts/certs/voltron.key` && \
		cat $< | sed "s;VOLTRON_CRT_BASE64;$$CERT64;" | sed "s;VOLTRON_KEY_BASE64;$$KEY64;" > $@
endif
	scripts/certs/clean-self-signed.sh scripts/certs

clean-manifests:
	rm -f manifests/voltron.yaml
	scripts/certs/clean-self-signed.sh scripts/certs


##########################################################################################
# TESTING 
##########################################################################################

GINKGO_ARGS += -cover -timeout 10m
GINKGO = ginkgo $(GINKGO_ARGS)

test: ut fv st

#############################################
# Run unit level tests
#############################################

.PHONY: ut
ut: CMD = go mod download && $(GINKGO) -r pkg/* internal/*
ut:
ifdef LOCAL
	$(CMD)
else
	$(DOCKER_GO_BUILD) sh -c '$(CMD)'
endif

#############################################
# Run package level functional level tests
#############################################
.PHONY: fv
fv: CMD = go mod download && $(GINKGO) -r test/fv
fv:
ifdef LOCAL
	$(CMD)
else
	$(DOCKER_GO_BUILD) sh -c '$(CMD)'
endif

#############################################
# Run system integration tests
#############################################
.PHONY: st
st: CMD = go mod download && $(GINKGO) -r test/st/
ifdef LOCAL
st: export TEST_CMD=$(CMD)
else
st: export TEST_CMD=$(DOCKER_GO_BUILD) sh -c '$(CMD)'
endif
st: $(COMPONENTS)
	sh test/st/run.sh

##########################################################################################
# CLEAN UP 
##########################################################################################
.PHONY: clean
clean: clean-build-image
	rm -rf $(BINDIR) docker-image/bin
	if test -d .go-pkg-cache; then chmod -R +w .go-pkg-cache && rm -rf .go-pkg-cache; fi
	find . -name "*.coverprofile" -type f -delete
	rm -rf docker-image/templates docker-image/scripts

clean-build-image:
	# Remove all variations e.g. tigera/voltron:latest + tigera/voltron:latest-amd64
	docker rmi -f $(BUILD_IMAGES) $(addsuffix :latest-$(ARCH), $(BUILD_IMAGES)) > /dev/null 2>&1 || true

###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
static-checks:
	docker run --rm -v $(CURDIR):/build-dir/$(PACKAGE_NAME):rw \
		-e LOCAL_USER_ID=$(MY_UID) \
		$(CALICO_BUILD) sh -c 'cd /build-dir/$(PACKAGE_NAME); ./static-checks.sh'

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed := $(shell ./install-git-hooks.sh)

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
ci: clean static-checks test images

#############################################
# Deploy images to registry
#############################################
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images push-images IMAGETAG=${BRANCH_NAME}

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

tag-images: imagetag
	ARCHES="$(ARCHES)" BUILD_IMAGES="$(BUILD_IMAGES)" PUSH_REPO="$(PUSH_REPO)" IMAGETAG="$(IMAGETAG)" make-helpers/tag-images

push-images: imagetag
	ARCHES="$(ARCHES)" BUILD_IMAGES="$(BUILD_IMAGES)" PUSH_REPO="$(PUSH_REPO)" IMAGETAG="$(IMAGETAG)" make-helpers/push-images

##########################################################################################
# LOCAL RUN
##########################################################################################

run-voltron:
	VOLTRON_TEMPLATE_PATH=docker-image/voltron/templates/guardian.yaml.tmpl VOLTRON_CERT_PATH=test go run cmd/voltron/main.go

run-guardian:
	GUARDIAN_VOLTRON_URL=127.0.0.1:5555 go run cmd/guardian/main.go

