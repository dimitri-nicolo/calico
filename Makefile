# Copyright 2019-20 Tigera Inc. All rights reserved.

APP_NAME        = voltron
PACKAGE_NAME   ?= github.com/tigera/$(APP_NAME)
GO_BUILD_VER   ?= v0.34
GIT_USE_SSH     = true
LOCAL_CHECKS    = mod-download

build: images

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

# Mount Semaphore configuration files.
ifdef ST_MODE
EXTRA_DOCKER_ARGS = -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp:rw -v /home/runner/config:/home/runner/config:rw -v /home/runner/docker_auth.json:/home/runner/config/docker_auth.json:rw
endif

include Makefile.common

##########################################################################################
# Define some constants
##########################################################################################
ARCHES = amd64
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
GIT_VERSION            = $(shell git describe --tags --dirty --always --long)

VERSION_FLAGS   = -X main.VERSION=$(BUILD_VERSION) \
                  -X main.BUILD_DATE=$(BUILD_BUILD_DATE) \
                  -X main.GIT_DESCRIPTION=$(BUILD_GIT_DESCRIPTION) \
                  -X main.GIT_REVISION=$(BUILD_GIT_REVISION)
BUILD_LDFLAGS   = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

#############################################
# Env vars related to packaging and releasing
#############################################
PUSH_REPO     ?= gcr.io/unique-caldron-775/cnx
COMPONENTS     = guardian voltron
BUILD_IMAGES  ?= $(addprefix tigera/, $(COMPONENTS))
RELEASE_BUILD ?= ""

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
	    sh -c '$(GIT_CONFIG_SSH) \
	        go build -o $@ -v $(LDFLAGS) cmd/$*/*.go && \
               ( ldd $@ 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $@ was not statically linked"; false ) )'

#############################################
# Docker Image
#############################################

## Build all app docker images.
images: manifests $(BUILD_IMAGES)
tigera/%: tigera/%-$(ARCH) ;
tigera/%-$(ARCH): $(BINDIR)/%-$(ARCH)
	rm -rf docker-image/$*/bin
	mkdir -p docker-image/$*/bin
	cp $(BINDIR)/$*-$(ARCH) docker-image/$*/bin/
	docker build --pull -t tigera/$*:latest-$(ARCH) ./docker-image/$*
ifeq ($(ARCH),amd64)
	docker tag tigera/$*:latest-$(ARCH) tigera/$*:latest
endif

MANIFESTS = docker-image/voltron/templates/guardian.yaml.tmpl

.PHONY: manifests
manifests: $(MANIFESTS)

.PHONY: docker-image/voltron/templates/guardian.yaml.tmpl
docker-image/voltron/templates/guardian.yaml.tmpl: manifests/guardian.yaml.tmpl
	mkdir -p docker-image/voltron/templates/
	sed -e "s;{{VOLTRON_DOCKER_PUSH_REPO}};$(PUSH_REPO);" \
	    -e "s;{{VOLTRON_DOCKER_TAG}};$(BRANCH_NAME);" $< > $@

clean-manifests:
	rm -f $(MANIFESTS)
	scripts/certs/clean-self-signed.sh scripts/certs

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
clean: clean-build-image clean-manifests
	rm -rf $(BINDIR) docker-image/bin
	find . -name "*.coverprofile" -type f -delete
	rm -rf docker-image/templates docker-image/scripts

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
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin

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
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images push-images IMAGETAG=${BRANCH_NAME}
	$(MAKE) tag-images push-images IMAGETAG=${GIT_VERSION}

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
# GO MODULES SURVIVAL
##########################################################################################

mod-graph:
	@if ! command -v dot >/dev/null; then echo "*** please install graphviz first"; exit 1; fi
	go mod why -m all | bash make-helpers/why2dot | sed -e 's/github.com\//@/g' | sort -u | bash make-helpers/dotify | dot -Tpng -o go.mod.png
	@echo "=> file://$$PWD/go.mod.png"

mod-tidy:
	go mod tidy
	go mod verify

mod-regen-sum:
	rm go.sum
	$(MAKE) -s mod-tidy

##########################################################################################
# LOCAL RUN
##########################################################################################

run-voltron:
	VOLTRON_TEMPLATE_PATH=docker-image/voltron/templates/guardian.yaml.tmpl VOLTRON_CERT_PATH=test go run cmd/voltron/main.go

run-guardian:
	GUARDIAN_VOLTRON_URL=127.0.0.1:5555 go run cmd/guardian/main.go

