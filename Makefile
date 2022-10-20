PACKAGE_NAME    ?= github.com/tigera/ingress-collector
GO_BUILD_VER    ?= v0.75
GIT_USE_SSH     := true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_INGRESS_COLLECTOR_PROJECT_ID)

INGRESS_COLLECTOR_IMAGE ?=tigera/ingress-collector
BUILD_IMAGES            ?=$(INGRESS_COLLECTOR_IMAGE)
DEV_REGISTRIES          ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES      ?=quay.io
RELEASE_BRANCH_PREFIX   ?=release-calient
DEV_TAG_SUFFIX          ?=calient-0.dev

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef CALICO_PATH
EXTRA_DOCKER_ARGS += -v $(CALICO_PATH):/go/src/github.com/projectcalico/calico/:ro
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH ?= $(GO_BUILD_VER)
MAKE_REPO   ?= https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

##############################################################################
PROTOC_VER ?= v0.1
PROTOC_CONTAINER ?= calico/protoc:$(PROTOC_VER)-$(BUILDARCH)

# Get version from git - used for releases.
INGRESS_GIT_VERSION ?= $(shell git describe --tags --dirty --always)
INGRESS_BUILD_DATE ?= $(shell date -u +'%FT%T%z')
INGRESS_GIT_REVISION ?= $(shell git rev-parse --short HEAD)
INGRESS_GIT_DESCRIPTION ?= $(shell git describe --tags)

ifeq ($(LOCAL_BUILD),true)
INGRESS_GIT_VERSION = $(shell git describe --tags --dirty --always)-dev-build
endif

VERSION_FLAGS=-X main.VERSION=$(INGRESS_GIT_VERSION) \
	-X main.BUILD_DATE=$(INGRESS_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(INGRESS_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(INGRESS_GIT_REVISION)
BUILD_LDFLAGS = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

############################################################################
# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	rm -rf report/ Makefile.common*
	rm -rf bin proto/felixbackend.pb.go
	-docker rmi $(INGRESS_COLLECTOR_IMAGE):latest-$(ARCH)
	-docker rmi $(INGRESS_COLLECTOR_IMAGE):$(VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
	-docker rmi $(INGRESS_COLLECTOR_IMAGE):latest
	-docker rmi $(INGRESS_COLLECTOR_IMAGE):$(VERSION)
endif

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/ingress-collector-,$(VALIDARCHES))

.PHONY: build
build: bin/ingress-collector-$(ARCH)

bin/ingress-collector-amd64: ARCH=amd64
bin/ingress-collector-arm64: ARCH=arm64
bin/ingress-collector-ppc64le: ARCH=ppc64le
bin/ingress-collector-s390x: ARCH=s390x
bin/ingress-collector-%: proto $(SRC_FILES)
ifndef VERSION
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LD_FLAGS))
endif
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build $(LDFLAGS) -v -o bin/ingress-collector-$(ARCH) \
	   	./cmd/ingress-collector'

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
PROTOC_MAPPINGS = Menvoy/api/v2/core/address.proto=github.com/envoyproxy/data-plane-api/envoy/api/v2/core,Menvoy/api/v2/core/base.proto=github.com/envoyproxy/data-plane-api/envoy/api/v2/core,Menvoy/type/http_status.proto=github.com/envoyproxy/data-plane-api/envoy/type,Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto,Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,Mgoogle/rpc/status.proto=github.com/gogo/googleapis/google/rpc

### Not used ATM
### This will be usable if we decide to use the latest protobuf dependencies.
ENVOYPROXY_GIT_URI:=github.com/envoyproxy/data-plane-api
PROTOC_GEN_VALIDATE_GIT_URI:=github.com/lyft/protoc-gen-validate
GOOGLEAPIS_GIT_URI:=github.com/gogo/googleapis
PROTOBUF_GIT_URI:=github.com/gogo/protobuf
UDPA_GIT_URI:=github.com/cncf/udpa

proto: proto/felixbackend.pb.go

proto/felixbackend.pb.go: proto/felixbackend.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              proto/*.proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):proto

###############################################################################
# Building the image
###############################################################################
CONTAINER_CREATED=.ingress-collector.created-$(ARCH)
.PHONY: image $(INGRESS_COLLECTOR_IMAGE)
image: $(INGRESS_COLLECTOR_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(INGRESS_COLLECTOR_IMAGE): $(CONTAINER_CREATED)
$(CONTAINER_CREATED): Dockerfile.$(ARCH) bin/ingress-collector-$(ARCH)
	docker build -t $(INGRESS_COLLECTOR_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(INGRESS_COLLECTOR_IMAGE):latest-$(ARCH) $(INGRESS_COLLECTOR_IMAGE):latest
endif
	touch $@

###############################################################################
# Managing the upstream library pins
###############################################################################
update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

## Update dependency pin
update-pins: guard-ssh-forwarding-bug proto/felixbackend.pb.go update-calico-pin

###############################################################################
## Perform static checks
# We need a custom static-checks to mount `.empty` instead of `deps` folder.
# Without this, golangci-lint fails to load

LINT_ARGS := --max-issues-per-linter 0 --max-same-issues 0 --timeout 5m --disable govet
.PHONY: static-checks-custom
static-checks-custom: mod-download
	$(DOCKER_RUN) -v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/deps \
	    $(CALICO_BUILD) golangci-lint run $(LINT_ARGS)

###############################################################################
# UTs
###############################################################################
WHAT?=.
GINKGO_FOCUS?=.*

.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev
ut: proto bin/ingress-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo -r --skipPackage deps,fv -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: fv
fv: proto bin/ingress-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo fv -r --skipPackage deps -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

###############################################################################
# CI
###############################################################################
.PHONY: ci
## Run what CI runs
ci:
	# Force ci to always run a clean before running other targets.
	# This is intentional so that irrespective of where ci is being run from
	# the ci target will always run a clean.
	# This change should be reverted when common Makefile runs the ci target
	# in a way that always runs the "clean" target.
	$(MAKE) clean build-all static-checks-custom ut

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: image-all cd-common
