# Copyright 2022 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/linseed

#############################################
# Env vars related to packaging and releasing
#############################################
LINSEED_IMAGE	?=linseed
BUILD_IMAGES?=$(LINSEED_IMAGE)

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
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')

# Exclude deprecation warnings (SA1019), since failing on deprecation defeats the purpose
# of deprecating.
LINT_ARGS += --exclude SA1019

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/linseed-$(ARCH)

.PHONY: bin/linseed-$(ARCH)
bin/linseed-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go'

###############################################################################
# BUILD IMAGE
###############################################################################
LINSEED_CONTAINER_CREATED=.linseed.created-$(ARCH)

# Build the docker image.
.PHONY: $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(LINSEED_IMAGE)

$(LINSEED_IMAGE): $(LINSEED_CONTAINER_CREATED)
$(LINSEED_CONTAINER_CREATED): docker-image/Dockerfile.$(ARCH) bin/linseed-$(ARCH)
	$(DOCKER_BUILD) -t $(LINSEED_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -rf bin \
		report/*.xml \
		config/
	find . -name '*.coverprofile' -type f -delete
	rm -f $(LINSEED_CREATED)
	-docker image rm -f $$(docker images $(LINSEED_IMAGE) -a -q)

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

st:
###############################################################################
# FV Tests
###############################################################################

run-image: stop-image
	# Run Linseed on Docker.
	docker run --detach \
	--net=host \
    --name=tigera-$(LINSEED_IMAGE) \
    -e "LINSEED_PORT=8444" \
    -v $(REPO_ROOT)/linseed/fv/cert/localhost.crt:/certs/https/tls.crt \
    -v $(REPO_ROOT)/linseed/fv/cert/localhost.key:/certs/https/tls.key \
    tigera/$(LINSEED_IMAGE):latest
    # Wait until Linseed is accepting requests.
	while ! docker ps -q --filter "name=tigera-$(LINSEED)"; do echo "Waiting for linseed to come up..."; sleep 2; done

stop-image:
	-docker rm -f $(shell docker ps -aq --filter "name=tigera-$(LINSEED_IMAGE)")

## Run the ginkgo FVs
fv: clean image run-image build-tests
	 $(DOCKER_RUN) \
	 	-e KUBECONFIG=/home/user/certs/kubeconfig \
		-v $(CERTS_PATH):/home/user/certs \
	 	-e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 $(CALICO_BUILD) \
		 sh -c '$(GIT_CONFIG_SSH) cd fv && ./fv.test $(GINKGO) -ginkgo.slowSpecThreshold 30'

# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
build-tests: $(shell find ./fv -type f -name '*.go' -print)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./fv -c --tags fvtests -o fv/fv.test'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut fv

## Deploys images to registry
cd: image-all cd-common
