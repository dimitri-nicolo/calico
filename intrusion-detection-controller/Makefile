##############################################################################
# Copyright 2019-2022 Tigera Inc. All rights reserved.
##############################################################################
include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/intrusion-detection-controller

IDS_CONTROLLER_IMAGE ?=intrusion-detection-controller
JOB_INSTALLER_IMAGE  ?=intrusion-detection-job-installer
BUILD_IMAGES         ?=$(IDS_CONTROLLER_IMAGE) $(JOB_INSTALLER_IMAGE)

TESLA ?= false
ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla/tigera
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-tesla
	DEV_TAG_SUFFIX        = tesla-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Define some constants
###############################################################################
BINDIR    ?= bin
SRC_FILES  = $(shell find pkg cmd -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../licensing/client -name '*.go') \
		$(shell find ../licensing/monitor -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')
TEST_DIRS ?= $(shell find pkg -name '*_test.go' -exec dirname {} \; | sort | uniq)
ifdef UNIT_TESTS
UNIT_TEST_FLAGS= -run $(UNIT_TESTS) -v
endif

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/version.VERSION=$(GIT_VERSION) \
	-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(DATE) \
	-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(GIT_COMMIT)

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "controller" instead of "$(BINDIR)/controller".
#########################################################################
IDS_CONTROLLER_CONTAINER_CREATED=.intrusion-detection-controller.created-$(ARCH)
JOB_INSTALLER_CONTAINER_CREATED=.intrusion-detection-job-installer.created-$(ARCH)

build: $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH) $(BINDIR)/intrusion-detection-job-installer-$(ARCH)

.PHONY: $(BINDIR)/controller-$(ARCH)
$(BINDIR)/controller-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/controller/controller.go'

.PHONY: $(BINDIR)/healthz-$(ARCH)
$(BINDIR)/healthz-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(BUILD_TAGS) cmd/healthz/healthz.go'

.PHONY: $(BINDIR)/intrusion-detection-job-installer-$(ARCH)
$(BINDIR)/intrusion-detection-job-installer-$(ARCH):
	$(DOCKER_GO_BUILD_CGO) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/install/install.go'

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(IDS_CONTROLLER_IMAGE) $(JOB_INSTALLER_IMAGE)

$(IDS_CONTROLLER_IMAGE): $(IDS_CONTROLLER_CONTAINER_CREATED)
$(IDS_CONTROLLER_CONTAINER_CREATED): docker-image/controller/Dockerfile.$(ARCH) $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	$(DOCKER_BUILD) -t $(IDS_CONTROLLER_IMAGE):latest-$(ARCH) -f docker-image/controller/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(IDS_CONTROLLER_IMAGE)
	touch $@

$(JOB_INSTALLER_IMAGE): $(JOB_INSTALLER_CONTAINER_CREATED)
$(JOB_INSTALLER_CONTAINER_CREATED): docker-image/install/Dockerfile.$(ARCH) $(BINDIR)/intrusion-detection-job-installer-$(ARCH)
	$(DOCKER_BUILD) -t $(JOB_INSTALLER_IMAGE):latest-$(ARCH) -f docker-image/install/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(JOB_INSTALLER_IMAGE)
	touch $@

##########################################################################
# Testing
##########################################################################
.PHONY: ut
ut: run-elastic run-ut stop-elastic

run-ut:
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: clean
clean:
	rm -fr $(BINDIR)
	rm -f $(IDS_CONTROLLER_CONTAINER_CREATED)
	rm -f $(JOB_INSTALLER_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(IDS_CONTROLLER_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(JOB_INSTALLER_IMAGE) -a -q)

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
    pkg/globalalert/elastic/Service \
    pkg/forwarder/LogDispatcher \

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
ci: clean static-checks ut

## Deploy images to registry
cd: image-all cd-common

###############################################################################
# Miscellaneous
###############################################################################
migrate-dashboards:
	bash migrate_dashboards.sh
