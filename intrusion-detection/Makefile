##############################################################################
# Copyright 2019-21 Tigera Inc. All rights reserved.
##############################################################################

PACKAGE_NAME   ?= github.com/tigera/intrusion-detection/controller
GO_BUILD_VER   ?= v0.65
GIT_USE_SSH     = true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_INTRUSION_DETECTION_PROJECT_ID)

ARCHES                ?=amd64

TESLA ?= false

IDS_IMAGE             ?=tigera/intrusion-detection-controller
JOB_INSTALLER_IMAGE   ?=tigera/intrusion-detection-job-installer
BUILD_IMAGES          ?=$(IDS_IMAGE) $(JOB_INSTALLER_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

ELASTIC_VERSION ?= 7.16.2

ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-tesla
	DEV_TAG_SUFFIX        = tesla-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

# Figure out the GID of the docker group so that we can set the user inside the
# container to be a member of that group. This, combined with mounting the
# docker socket, allows the UTs to start docker containers.
MY_DOCKER_GID=$(shell getent group docker | cut -d: -f3)

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/* -e EXTRA_GROUP_ID=$(MY_DOCKER_GID) -v /var/run/docker.sock:/var/run/docker.sock:rw

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

###############################################################################
# Define some constants
###############################################################################
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
UNIT_TEST_FLAGS= -run $(UNIT_TESTS) -v
endif

CONTROLLER_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
CONTROLLER_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CONTROLLER_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CONTROLLER_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(CONTROLLER_VERSION) \
	-X main.BUILD_DATE=$(CONTROLLER_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(CONTROLLER_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(CONTROLLER_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "controller" instead of "$(BINDIR)/controller".
#########################################################################
build: $(BUILD_IMAGES)

controller: $(BINDIR)/controller

$(BINDIR)/controller: $(BINDIR)/controller-amd64
	cd $(BINDIR) && ln -sf controller-$(ARCH) controller

$(BINDIR)/controller-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building controller...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(LDFLAGS) $(BUILD_TAGS) "$(PACKAGE_NAME)/cmd/controller" && \
               ( ldd $(BINDIR)/controller-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
	             ( echo "Error: $(BINDIR)/controller-$(ARCH) was not statically linked"; false ) )'

healthz: $(BINDIR)/healthz

$(BINDIR)/healthz: $(BINDIR)/healthz-amd64
	cd $(BINDIR) && ln -sf healthz-$(ARCH) healthz

$(BINDIR)/healthz-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building healthz...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/healthz" && \
               ( ldd $(BINDIR)/healthz-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
	             ( echo "Error: $(BINDIR)/healthz-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(IDS_IMAGE) $(IDS_IMAGE)-$(ARCH) $(JOB_INSTALLER_IMAGE) $(JOB_INSTALLER_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-images-,$(ARCHES))
sub-images-%:
	$(MAKE) images ARCH=$*

images: $(BUILD_IMAGES)

$(IDS_IMAGE): $(IDS_IMAGE)-$(ARCH)
$(IDS_IMAGE)-$(ARCH): $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	rm -rf docker-image/controller/bin
	mkdir -p docker-image/controller/bin
	cp $(BINDIR)/controller-$(ARCH) docker-image/controller/bin/
	cp $(BINDIR)/healthz-$(ARCH) docker-image/controller/bin/
	docker build --pull -t $(IDS_IMAGE):latest-$(ARCH) --file ./docker-image/controller/Dockerfile.$(ARCH) docker-image/controller
ifeq ($(ARCH),amd64)
	docker tag $(IDS_IMAGE):latest-$(ARCH) $(IDS_IMAGE):latest
endif

$(JOB_INSTALLER_IMAGE): $(JOB_INSTALLER_IMAGE)-$(ARCH)
$(JOB_INSTALLER_IMAGE)-$(ARCH):
	# Run from the "install" sub-directory so that docker has access to all the python bits.
	docker build --pull -t $(JOB_INSTALLER_IMAGE):latest-$(ARCH) --build-arg version=$(BUILD_VERSION) --file ./docker-image/install/Dockerfile.$(ARCH) install
ifeq ($(ARCH),amd64)
	docker tag $(JOB_INSTALLER_IMAGE):latest-$(ARCH) $(JOB_INSTALLER_IMAGE):latest
endif

##########################################################################
# Testing
##########################################################################
.PHONY: ut
ut: run-elastic run-ut stop-elastic

run-ut:
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv st
fv:
	echo "FV not implemented yet"
st:
	echo "ST not implemented yet"

.PHONY: clean
clean: clean-bin clean-build-images
	rm -rf vendor Makefile.common*
clean-build-images:
	docker rmi -f $(IDS_IMAGE) > /dev/null 2>&1 || true
	docker rmi -f $(JOB_INSTALLER_IMAGE) > /dev/null 2>&1 || true

clean-bin:
	rm -rf $(BINDIR) \
			docker-image/controller/bin

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
    pkg/globalalert/elastic/Service \
    pkg/forwarder/LogDispatcher \

## Run elasticsearch as a container (tigera-elastic)
run-elastic: stop-elastic
	# Run ES on Docker.
	docker run --detach \
	--net=host \
	--name=tigera-elastic \
	-e "discovery.type=single-node" \
	docker.elastic.co/elasticsearch/elasticsearch:$(ELASTIC_VERSION)

	# Wait until ES is accepting requests.
	@while ! docker exec tigera-elastic curl localhost:9200 2> /dev/null; do echo "Waiting for Elasticsearch to come up..."; sleep 2; done


## Stop elasticsearch with name tigera-elastic
stop-elastic:
	-docker rm -f tigera-elastic

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
ci: clean check-fmt test images-all

## Deploy images to registry
cd: images cd-common

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
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

update-pins: guard-ssh-forwarding-bug update-calico-pin

###############################################################################
# Miscellaneous
###############################################################################
migrate-dashboards:
	bash install/migrate_dashboards.sh
