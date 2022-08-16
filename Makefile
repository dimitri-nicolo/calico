ORGANIZATION=tigera
PACKAGE_NAME?=github.com/tigera/egress-gateway
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_EGRESS_GATEWAY_PROJECT_ID)

GO_BUILD_VER?=v0.73
GIT_USE_SSH=true

EGRESS_GATEWAY_IMAGE  ?=tigera/egress-gateway
EGRESS_GATEWAY_TESTING_IMAGE ?=tigera/egress-gateway-testing
BUILD_IMAGES          ?=$(EGRESS_GATEWAY_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

EXTRA_DOCKER_ARGS+=-e GOPRIVATE='github.com/tigera/*'

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
SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
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
# Compile Proto's
###############################################################################
# We use gogofast for protobuf compilation.  Regular gogo is incompatible with
# gRPC, since gRPC uses golang/protobuf for marshalling/unmarshalling in that
# case.  See https://github.com/gogo/protobuf/issues/386 for more details.
# Note that we cannot seem to use gogofaster because of incompatibility with
# Envoy's validation library.
# When importing, we must use gogo versions of google/protobuf and
# google/rpc (aka googleapis).
PROTOC_IMPORTS =  -I proto\
		  -I ./


proto: proto/felixbackend.pb.go

# Generate the protobuf bindings for go. The proto/felixbackend.pb.go file is included in SRC_FILES
# In order to make use of complex structures like protobuf.google.timestamp, we need to link the
# protobuf google types when generating go code from protobuf messages
proto/felixbackend.pb.go: proto/felixbackend.proto
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		  -v $(CURDIR):/code -v $(CURDIR)/proto:/src:rw \
		      $(PROTOC_CONTAINER) \
		      --gogofaster_out=\
	Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
	plugins=grpc:. \
	felixbackend.proto
	# Make sure the generated code won't cause a static-checks failure.
	$(MAKE) fix
###############################################################################
# Building the binary
###############################################################################
# If local build is set, then always build the binary since we might not
# detect when another local repository has been modified.
ifeq ($(LOCAL_BUILD),true)
.PHONY: $(SRC_FILES)
endif

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
.PHONY:local_build
ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go:rw
local_build:
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go

else
local_build:
	@echo "Building Egress Gateway for Calico version ${GIT_VERSION}"
endif
EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

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
bin/egressd-%: local_build proto $(SRC_FILES)
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
	docker build --pull -t $(EGRESS_GATEWAY_IMAGE):latest-$(ARCH) . --build-arg GIT_VERSION=$(GIT_VERSION) -f ./Dockerfile.$(ARCH)
	touch $@

$(EGRESS_GATEWAY_TESTING_IMAGE): $(GATEWAY_TEST_CONTAINER_CREATED)
$(GATEWAY_TEST_CONTAINER_CREATED): register ./test/Dockerfile.$(ARCH) $(GATEWAY_CONTAINER_FILES) $(GATEWAY_DAEMON_EXECUTABLE)
	docker build --pull -t $(EGRESS_GATEWAY_TESTING_IMAGE):latest-$(ARCH) . --build-arg GIT_VERSION=$(GIT_VERSION) -f ./test/Dockerfile.$(ARCH)
	touch $@

## Run the tests in a container. Useful for CI, Mac dev
ut: local_build proto
	mkdir -p report
	$(DOCKER_RUN) $(CALICO_BUILD) /bin/bash -c "$(GIT_CONFIG_SSH) go test -v $(GOTEST_ARGS) ./..."


###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean image-all
## Deploys images to registry
cd: cd-common
