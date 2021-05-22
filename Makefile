PACKAGE_NAME?=github.com/projectcalico/app-policy
GO_BUILD_VER?=v0.53

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_APP_POLICY_PRIVATE_PROJECT_ID)

###############################################################################
PROTOC_VER?=v0.1
PROTOC_CONTAINER?=calico/protoc:$(PROTOC_VER)-$(BUILDARCH)

DIKASTES_GIT_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)

# Get version from git - used for releases.
GIT_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
ifeq ($(LOCAL_BUILD),true)
	GIT_VERSION = $(shell git describe --tags --dirty --always --abbrev=12)-dev-build
endif

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
LOCAL_USER_ID:=$(shell id -u)
MY_GID:=$(shell id -g)

SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

# If local build is set, then always build the binary since we might not
# detect when another local repository has been modified.
ifeq ($(LOCAL_BUILD),true)
.PHONY: $(SRC_FILES)
endif

LIBCALICO_REPO   = github.com/tigera/libcalico-go-private

GIT_USE_SSH?=true

DIKASTES_IMAGE        ?=tigera/dikastes
BUILD_IMAGES          ?= $(DIKASTES_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev
# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
.PHONY:local_build

ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go:rw
local_build:
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
else
local_build:
	@echo "Building app-policy-private"
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

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

# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	rm -rf .go-pkg-cache report vendor bin Makefile.common*
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	-docker rmi $(DIKASTES_IMAGE):latest-$(ARCH)
	-docker rmi $(DIKASTES_IMAGE):$(VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
	-docker rmi $(DIKASTES_IMAGE):latest
	-docker rmi $(DIKASTES_IMAGE):$(VERSION)
endif

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/dikastes-,$(VALIDARCHES))

.PHONY: build
## Build the binary for the current architecture and platform
build: bin/dikastes-$(ARCH) bin/healthz-$(ARCH)

bin/dikastes-amd64: ARCH=amd64
bin/dikastes-arm64: ARCH=arm64
bin/dikastes-ppc64le: ARCH=ppc64le
bin/dikastes-s390x: ARCH=s390x
bin/dikastes-%: local_build proto $(SRC_FILES)
	mkdir -p bin
	$(DOCKER_RUN_RO) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_FLAGS) -ldflags "-X main.VERSION=$(DIKASTES_GIT_VERSION) -s -w" -v -o bin/dikastes-$(ARCH) ./cmd/dikastes'

bin/healthz-amd64: ARCH=amd64
bin/healthz-arm64: ARCH=arm64
bin/healthz-ppc64le: ARCH=ppc64le
bin/healthz-s390x: ARCH=s390x
bin/healthz-%: local_build proto $(SRC_FILES)
	mkdir -p bin || true
	-mkdir -p .go-pkg-cache $(GOMOD_CACHE) || true
	$(DOCKER_RUN_RO) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_FLAGS) -ldflags "-X main.VERSION=$(DIKASTES_GIT_VERSION) -s -w" -v -o bin/healthz-$(ARCH) ./cmd/healthz'

# We use gogofast for protobuf compilation.  Regular gogo is incompatible with
# gRPC, since gRPC uses golang/protobuf for marshalling/unmarshalling in that
# case.  See https://github.com/gogo/protobuf/issues/386 for more details.
# Note that we cannot seem to use gogofaster because of incompatibility with
# Envoy's validation library.
# When importing, we must use gogo versions of google/protobuf and
# google/rpc (aka googleapis).
PROTOC_IMPORTS =  -I proto\
		  -I ./
# Also remap the output modules to gogo versions of google/protobuf and google/rpc
PROTOC_MAPPINGS = Menvoy/api/v2/core/address.proto=github.com/envoyproxy/data-plane-api/envoy/api/v2/core,Menvoy/api/v2/core/base.proto=github.com/envoyproxy/data-plane-api/envoy/api/v2/core,Menvoy/type/http_status.proto=github.com/envoyproxy/data-plane-api/envoy/type,Menvoy/type/percent.proto=github.com/envoyproxy/data-plane-api/envoy/type,Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto,Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,Mgoogle/rpc/status.proto=github.com/gogo/googleapis/google/rpc,Menvoy/service/auth/v2/external_auth.proto=github.com/envoyproxy/data-plane-api/envoy/service/auth/v2

proto: proto/felixbackend.pb.go proto/healthz.pb.go

proto/felixbackend.pb.go: proto/felixbackend.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
		      $(PROTOC_CONTAINER) \
		      $(PROTOC_IMPORTS) \
		      proto/*.proto \
		      --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):proto

proto/healthz.pb.go: proto/healthz.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
		      $(PROTOC_CONTAINER) \
		      $(PROTOC_IMPORTS) \
		      proto/*.proto \
		      --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):proto

###############################################################################
# Building the image
###############################################################################
CONTAINER_CREATED=.dikastes.created-$(ARCH)
.PHONY: image $(DIKASTES_IMAGE)
image: $(DIKASTES_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(DIKASTES_IMAGE): $(CONTAINER_CREATED)
$(CONTAINER_CREATED): Dockerfile.$(ARCH) bin/dikastes-$(ARCH) bin/healthz-$(ARCH)
	docker build -t $(DIKASTES_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(DIKASTES_IMAGE):latest-$(ARCH) $(DIKASTES_IMAGE):latest
endif
	touch $@

update-pins: proto/felixbackend.pb.go proto/healthz.pb.go replace-libcalico-pin

###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev
ut: local_build proto
	mkdir -p report
	$(DOCKER_RUN) $(CALICO_BUILD) /bin/bash -c "$(GIT_CONFIG_SSH) go test -v $(GINKGO_ARGS) ./... | go-junit-report > ./report/tests.xml"

.PHONY: ci
ci: mod-download build-all check-generated-files static-checks ut

## Check if generated files are out of date
.PHONY: check-generated-files
check-generated-files: proto
	if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated files are out of date."; \
	  false; \
	else \
	  echo "Generated files are up to date."; \
	fi

## Avoid unplanned go.sum updates
.PHONY: undo-go-sum check-dirty
undo-go-sum:
	@if (git status --porcelain go.sum | grep -o 'go.sum'); then \
	  echo "Undoing go.sum update..."; \
	  git checkout -- go.sum; \
	fi

## Check if generated image is dirty
check-dirty: undo-go-sum
	@if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated image is dirty:"; \
	  git status --porcelain; \
	  false; \
	fi

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: image-all check-dirty cd-common