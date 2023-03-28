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

# We use -X to insert the version information into the placeholder variables
# in the version package.
LDFLAGS = -X $(PACKAGE_NAME)/pkg/config/version.BuildVersion=$(GIT_VERSION) \
			-X $(PACKAGE_NAME)/pkg/config/version.BuildDate=$(DATE) \
			-X $(PACKAGE_NAME)/pkg/config/version.GitDescription=$(GIT_DESCRIPTION) \
			-X $(PACKAGE_NAME)/pkg/config/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/linseed-$(ARCH)

.PHONY: bin/linseed-$(ARCH)
bin/linseed-$(ARCH): $(SRC_FILES)
	$(call build_static_cgo_boring_binary, cmd/$*/*.go, $@)

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
clean: stop-elastic
	rm -rf bin \
		report/*.xml \
		config/
	find . -name '*.coverprofile' -type f -delete
	rm -f $(LINSEED_CREATED)
	-docker image rm -f $$(docker images $(LINSEED_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
MOCKERY_FILE_PATHS= \
	pkg/backend/api/FlowBackend \
	pkg/backend/api/FlowLogBackend \
	pkg/backend/api/L7FlowBackend \
	pkg/backend/api/L7LogBackend \
	pkg/backend/api/DNSFlowBackend \
	pkg/backend/api/DNSLogBackend \
	pkg/backend/api/EventsBackend \
	pkg/backend/api/AuditBackend \
	pkg/backend/api/BGPBackend \
	pkg/backend/api/ProcessBackend \
	pkg/backend/api/WAFBackend

#############################################
# Run unit level tests
#############################################

.PHONY: ut
## Run only Unit Tests.
ut: stop-elastic run-elastic ut-fast
ut-fast:
	# We run the tests 5 times to try and weed out test flakes.
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./... -cover -count 5'

st:
	@echo "No STs"

###############################################################################
# FV Tests
###############################################################################

# Run Linseed as a local Docker container
run-image: stop-image run-elastic
	docker run --detach \
		--net=host \
		--name=tigera-$(LINSEED_IMAGE) \
		-v $(REPO_ROOT)/linseed/fv/cert/localhost.crt:/certs/https/tls.crt \
		-v $(REPO_ROOT)/linseed/fv/cert/localhost.key:/certs/https/tls.key \
		-v $(REPO_ROOT)/linseed/fv/cert/RootCA.crt:/certs/https/client.crt \
		-e ELASTIC_HOST="localhost" \
		-e ELASTIC_SCHEME="http" \
		-e LINSEED_LOG_LEVEL="debug" \
		tigera/$(LINSEED_IMAGE):latest
	# Wait until Linseed is accepting requests on its readiness port.
	while ! curl -k http://localhost:8080/readiness; do echo "Waiting for linseed to come up..."; sleep 2; done

stop-image:
	-docker rm -f $(shell docker ps -aq --filter "name=tigera-$(LINSEED_IMAGE)")

## Run the FVs
fv: clean image build-tests run-image
	 $(DOCKER_RUN) \
	 	-e KUBECONFIG=/home/user/certs/kubeconfig \
	 	$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) cd fv && ./fv.test'

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
ci: clean static-checks test

## Deploys images to registry
cd: image-all cd-common
