# Copyright 2022 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/continuous-policy-recommendation

#############################################
# Env vars related to packaging and releasing
#############################################
POLICY_RECOMMENDATION_IMAGE ?=continuous-policy-recommendation
BUILD_IMAGES        ?=$(POLICY_RECOMMENDATION_IMAGE)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES = $(shell find pkg cmd -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/continuous-policy-recommendation-$(ARCH)

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

.PHONY: bin/continuous-policy-recommendation-$(ARCH)
bin/continuous-policy-recommendation-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go'

###############################################################################
# BUILD IMAGE
###############################################################################
POLICY_RECOMMENDATION_CONTAINER_CREATED=.continuous-policy-recommendation.created-$(ARCH)

# Build the docker image.
.PHONY: $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(BUILD_IMAGES)

$(POLICY_RECOMMENDATION_IMAGE): $(POLICY_RECOMMENDATION_CONTAINER_CREATED)
$(POLICY_RECOMMENDATION_CONTAINER_CREATED): docker-image/Dockerfile.$(ARCH) bin/continuous-policy-recommendation-$(ARCH)
	$(DOCKER_BUILD) -t $(POLICY_RECOMMENDATION_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -rf bin \
	rm -f $(POLICY_RECOMMENDATION_CONTAINER_CREATED)
	find . -name "*.coverprofile" -type f -delete
	-docker image rm -f $$(docker images $(POLICY_RECOMMENDATION_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################

.PHONY: ut
## Run only Unit Tests.
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) $(GINKGO) -r pkg/*'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut

## Deploys images to registry
cd: image-all cd-common
