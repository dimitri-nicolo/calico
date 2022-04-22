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
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla
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
BINDIR        ?= bin
SRC_FILES      = $(shell find pkg cmd -name '*.go') $(shell find ../libcalico-go/lib -name '*.go') $(shell find ../lma/pkg -name '*.go')
TEST_DIRS     ?= $(shell find pkg -name '*_test.go' -exec dirname {} \\; | sort | uniq)
ifdef UNIT_TESTS
UNIT_TEST_FLAGS= -run $(UNIT_TESTS) -v
endif

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/version.VERSION=$(GIT_VERSION) \
	-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(DATE) \
	-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(GIT_COMMIT)

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "controller" instead of "$(BINDIR)/controller".
#########################################################################
build: $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH)

.PHONY: $(BINDIR)/controller-$(ARCH)
$(BINDIR)/controller-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/controller/controller.go && \
               ( ldd $(BINDIR)/controller-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
	             ( echo "Error: $(BINDIR)/controller-$(ARCH) was not statically linked"; false ) )'

.PHONY: $(BINDIR)/healthz-$(ARCH)
$(BINDIR)/healthz-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(BUILD_TAGS) cmd/healthz/healthz.go && \
               ( ldd $(BINDIR)/healthz-$(ARCH) 2>&1 | \
			       grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
	             ( echo "Error: $(BINDIR)/healthz-$(ARCH) was not statically linked"; false ) )'

# by default, build the image for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-images-,$(ARCHES))
sub-images-%:
	$(MAKE) images ARCH=$*

images: $(BUILD_IMAGES)
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

$(IDS_CONTROLLER_IMAGE): $(IDS_CONTROLLER_IMAGE)-$(ARCH)
$(IDS_CONTROLLER_IMAGE)-$(ARCH): $(BINDIR)/controller-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	$(DOCKER_BUILD) -t $(IDS_CONTROLLER_IMAGE):latest-$(ARCH) -f docker-image/controller/Dockerfile.$(ARCH) . --load

$(JOB_INSTALLER_IMAGE): $(JOB_INSTALLER_IMAGE)-$(ARCH)
$(JOB_INSTALLER_IMAGE)-$(ARCH):
	# Run from the "install" sub-directory so that docker has access to all the python bits.
	$(DOCKER_BUILD) -t $(JOB_INSTALLER_IMAGE):latest-$(ARCH) -f docker-image/install/Dockerfile.$(ARCH) install --load

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
ci: clean check-fmt test images-all

## Deploy images to registry
cd: images cd-common

###############################################################################
# Miscellaneous
###############################################################################
migrate-dashboards:
	bash install/migrate_dashboards.sh
