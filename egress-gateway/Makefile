include ../metadata.mk

PACKAGE_NAME?=github.com/projectcalico/calico/egress-gateway
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_EGRESS_GATEWAY_PROJECT_ID)

EGRESS_GATEWAY_IMAGE  ?=egress-gateway
EGRESS_GATEWAY_TESTING_IMAGE ?=egress-gateway-testing
BUILD_IMAGES          ?=$(EGRESS_GATEWAY_IMAGE)
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

###############################################################################
# Shortcut targets
###############################################################################
default: build image
test: ut ## Run the tests for the current platform/architecture

###############################################################################
# Variables controlling the image
###############################################################################
GATEWAY_CONTAINER_CREATED=.egress_gateway.created-$(ARCH)
GATEWAY_TEST_CONTAINER_CREATED=.egress_gateway_testing.created-$(ARCH)
# Files that go into the image
GATEWAY_CONTAINER_FILES=$(shell find ./filesystem -type f)
GATEWAY_DAEMON_EXECUTABLE=./bin/egressd-$(ARCH)
# Files to be built
SRC_FILES=$(shell find . -name '*.go' | grep -v vendor) ../felix/proto/felixbackend.pb.go $(shell find ../libcalico-go -name '*.go' | grep -v vendor)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
## Clean enough that a new release build will be clean
###############################################################################
.PHONY: clean
clean:
	find . -name '*.created' -exec rm -f {} +
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf .go-pkg-cache bin Makefile.common*
	rm -rf certs *.tar
	rm -rf dist
	rm -rf Makefile.common*
	# Delete images that we built in this repo
	-docker rmi $(EGRESS_GATEWAY_IMAGE):latest-$(ARCH)
	-docker rmi $(EGRESS_GATEWAY_TESTING_IMAGE):latest-$(ARCH)

###############################################################################
# Protobufs, defer to felix's makefile.
###############################################################################

../felix/proto/felixbackend.pb.go: ../felix/proto/felixbackend.proto
	$(MAKE) --directory ../felix protobuf

###############################################################################
# Building the binary
###############################################################################

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/egressd-,$(VALIDARCHES))

.PHONY: build
## Build the binary for the current architecture and platform
build: bin/egressd-$(ARCH)
bin/egressd-amd64: ARCH=amd64
bin/egressd-%: $(SRC_FILES)
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e CGO_LDFLAGS=$(CGO_LDFLAGS) $(CALICO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
		  go build $(BUILD_FLAGS) -ldflags "-X main.VERSION=$(GIT_VERSION) " -o bin/egressd-$(ARCH) ./cmd/egressd'

###############################################################################
# Building the image
###############################################################################
## Create the image for the current ARCH
image: $(EGRESS_GATEWAY_IMAGE)
image-testing: $(EGRESS_GATEWAY_TESTING_IMAGE)

## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(EGRESS_GATEWAY_IMAGE): $(GATEWAY_CONTAINER_CREATED)
$(GATEWAY_CONTAINER_CREATED): register ./Dockerfile.$(ARCH) $(GATEWAY_CONTAINER_FILES) $(GATEWAY_DAEMON_EXECUTABLE)
	$(DOCKER_BUILD) -t $(EGRESS_GATEWAY_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

$(EGRESS_GATEWAY_TESTING_IMAGE): $(GATEWAY_TEST_CONTAINER_CREATED)
$(GATEWAY_TEST_CONTAINER_CREATED): register ./test/Dockerfile.$(ARCH) $(GATEWAY_CONTAINER_FILES) $(GATEWAY_DAEMON_EXECUTABLE)
	$(DOCKER_BUILD) -t $(EGRESS_GATEWAY_TESTING_IMAGE):latest-$(ARCH) -f test/Dockerfile.$(ARCH) . --load
	touch $@

## Run the tests in a container. Useful for CI, Mac dev
ut: ../felix/proto/felixbackend.pb.go
	mkdir -p report
	$(DOCKER_RUN) $(CALICO_BUILD) /bin/bash -c "$(GIT_CONFIG_SSH) go test -v $(GOTEST_ARGS) ./..."

fv:
	echo "egress-gateway currently has no FV tests."

st:
	echo "egress-gateway ST tests are currently in the node 'k8st' framework."
	echo "To run them: make --directory ../node egress-ip-tests"

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean image-all test
## Deploys images to registry
cd: cd-common
