# Copyright 2019-2022 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME   ?= github.com/projectcalico/calico/voltron

#############################################
# Env vars related to packaging and releasing
#############################################
COMPONENTS            ?=guardian voltron
VOLTRON_IMAGE         ?=voltron
VOLTRON_FIPS_IMAGE    ?=voltron-fips
GUARDIAN_IMAGE        ?=guardian
BUILD_IMAGES          ?=$(VOLTRON_IMAGE)

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

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

BINDIR        ?= bin
SRC_FILES = $(shell find . -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib/logutils -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')
TEST_DIRS     ?= $(shell find pkg -name '*_test.go' -exec dirname {} \; | sort -u)
ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

#############################################
# Env vars related to building
#############################################

# Flags for building the binaries.
#
# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT) \

##########################################################################################
# BUILD
##########################################################################################
build: $(BINDIR)/voltron-$(ARCH) $(BINDIR)/guardian-$(ARCH)

.PHONY: $(BINDIR)/voltron-$(ARCH)
$(BINDIR)/voltron-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/voltron/main.go'

.PHONY: $(BINDIR)/poc-fips
$(BINDIR)/poc-fips:
	$(DOCKER_RUN) -e CGO_ENABLED=1 $(CALICO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/tmp-fips/fips.go'

.PHONY: $(BINDIR)/voltron-fips-$(ARCH)
$(BINDIR)/voltron-fips-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) -e CGO_ENABLED=1 $(CALICO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/voltron/main.go'


.PHONY: $(BINDIR)/guardian-$(ARCH)
$(BINDIR)/guardian-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v -ldflags "$(VERSION_FLAGS)" $(BUILD_TAGS) cmd/guardian/main.go'

#############################################
# Docker Image
#############################################
VOLTRON_CONTAINER_CREATED=.voltron.created-$(ARCH)
GUARDIAN_CONTAINER_CREATED=.guardian.created-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(BUILD_IMAGES)

$(VOLTRON_IMAGE): $(VOLTRON_CONTAINER_CREATED)
$(VOLTRON_CONTAINER_CREATED): docker-image/voltron/Dockerfile.$(ARCH) $(BINDIR)/voltron-$(ARCH) $(BINDIR)/voltron-fips-$(ARCH)
	$(DOCKER_BUILD) -t $(VOLTRON_IMAGE):latest-$(ARCH) --build-arg BINARY=bin/voltron-$(ARCH) -f docker-image/voltron/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(VOLTRON_IMAGE)
	touch $@
ifeq ($(ARCH), amd64)
	$(DOCKER_BUILD) -t $(VOLTRON_FIPS_IMAGE):latest-$(ARCH) --build-arg BINARY="bin/voltron-fips-$(ARCH)" -f docker-image/voltron/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(VOLTRON_FIPS_IMAGE)
	touch $@
endif

.PHONY: image2
image2: $(BINDIR)/voltron-fips-$(ARCH)
	$(DOCKER_BUILD) -t $(VOLTRON_FIPS_IMAGE):latest-$(ARCH) --build-arg BINARY="bin/voltron-fips-$(ARCH)" -f docker-image/voltron/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(VOLTRON_FIPS_IMAGE)
	touch $@


$(GUARDIAN_IMAGE): $(GUARDIAN_CONTAINER_CREATED)
$(GUARDIAN_CONTAINER_CREATED): docker-image/guardian/Dockerfile.$(ARCH) $(BINDIR)/guardian-$(ARCH)
	$(DOCKER_BUILD) -t $(GUARDIAN_IMAGE):latest-$(ARCH) -f docker-image/guardian/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(GUARDIAN_IMAGE)
	touch $@

##########################################################################################
# CLEAN UP
##########################################################################################
.PHONY: clean
clean:
	rm -rf $(BINDIR)
	find . -name "*.coverprofile" -type f -delete
	rm -f $(VOLTRON_CONTAINER_CREATED)
	rm -f $(GUARDIAN_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(VOLTRON_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(GUARDIAN_IMAGE) -a -q)

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
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(GINKGO) -r pkg/* internal/*'

#############################################
# Run package level functional level tests
#############################################
.PHONY: fv
## Run only Package Functional Tests.
fv:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(GINKGO) -r test/fv'

##########################################################################################
# CI/CD
##########################################################################################
.PHONY: ci cd

#############################################
# Run CI cycle - build, test, etc.
#############################################
## Run all CI steps for build and test, likely other targets.
ci: clean static-checks ut fv

#############################################
# Deploy images to registry
#############################################
## Run all CD steps, normally pushing images out to registries.
cd: image-all cd-common

##########################################################################################
# MOCKING
##########################################################################################
# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS=\
	pkg/tunnel/Dialer
