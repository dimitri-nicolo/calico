PACKAGE_NAME    ?= github.com/tigera/l7-collector
GO_BUILD_VER    ?= v0.53
GIT_USE_SSH     := true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_L7_COLLECTOR_PROJECT_ID)

L7_COLLECTOR_IMAGE    ?=tigera/l7-collector
ENVOY_INIT_IMAGE      ?=tigera/envoy-init
BUILD_IMAGES          ?=$(L7_COLLECTOR_IMAGE) $(ENVOY_INIT_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

##############################################################################
PROTOC_VER ?= v0.1
PROTOC_CONTAINER ?= calico/protoc:$(PROTOC_VER)-$(BUILDARCH)

# Get version from git - used for releases.
ENVOY_COLLECTOR_GIT_VERSION ?= $(shell git describe --tags --dirty --always --abbrev=12)
ENVOY_COLLECTOR_BUILD_DATE ?= $(shell date -u +'%FT%T%z')
ENVOY_COLLECTOR_GIT_REVISION ?= $(shell git rev-parse --short HEAD)
ENVOY_COLLECTOR_GIT_DESCRIPTION ?= $(shell git describe --tags --abbrev=12 || echo '<unknown>')

ifeq ($(LOCAL_BUILD),true)
ENVOY_COLLECTOR_GIT_VERSION = $(shell git describe --tags --dirty --always --abbrev=12)-dev-build
endif

VERSION_FLAGS=-X main.VERSION=$(ENVOY_COLLECTOR_GIT_VERSION) \
	-X main.BUILD_DATE=$(ENVOY_COLLECTOR_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(ENVOY_COLLECTOR_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(ENVOY_COLLECTOR_GIT_REVISION)
BUILD_LDFLAGS = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

ENVOY_API=deps/github.com/envoyproxy/data-plane-api
EXT_AUTH=$(ENVOY_API)/envoy/service/auth/v2alpha/
ADDRESS=$(ENVOY_API)/envoy/api/v2/core/address
V2_BASE=$(ENVOY_API)/envoy/api/v2/core/base
HTTP_STATUS=$(ENVOY_API)/envoy/type/http_status

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

############################################################################
# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	rm -rf report/
	rm -rf bin proto/felixbackend.pb.go
	-docker rmi $(L7_COLLECTOR_IMAGE):latest-$(ARCH)
	-docker rmi $(L7_COLLECTOR_IMAGE):$(VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
	-docker rmi $(L7_COLLECTOR_IMAGE):latest
	-docker rmi $(L7_COLLECTOR_IMAGE):$(VERSION)
endif

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/l7-collector-,$(VALIDARCHES))

.PHONY: build
build: bin/l7-collector-$(ARCH)

bin/l7-collector-amd64: ARCH=amd64
bin/l7-collector-arm64: ARCH=arm64
bin/l7-collector-ppc64le: ARCH=ppc64le
bin/l7-collector-s390x: ARCH=s390x
bin/l7-collector-%: proto $(SRC_FILES)
ifndef VERSION
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LD_FLAGS))
endif
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build $(LDFLAGS) -v -o bin/l7-collector-$(ARCH) \
	   	./cmd/l7-collector'

# We use gogofast for protobuf compilation.  Regular gogo is incompatible with
# gRPC, since gRPC uses golang/protobuf for marshalling/unmarshalling in that
# case.  See https://github.com/gogo/protobuf/issues/386 for more details.
# Note that we cannot seem to use gogofaster because of incompatibility with
# Envoy's validation library.
# When importing, we must use gogo versions of google/protobuf and
# google/rpc (aka googleapis).
PROTOC_IMPORTS =  -I $(ENVOY_API) \
                  -I deps/github.com/gogo/protobuf/protobuf \
                  -I deps/github.com/gogo/protobuf \
                  -I deps/github.com/lyft/protoc-gen-validate\
                  -I deps/github.com/gogo/googleapis\
                  -I proto\
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

proto-download:
	mkdir -p deps
	-git clone https://$(ENVOYPROXY_GIT_URI).git deps/$(ENVOYPROXY_GIT_URI)
	-git clone https://$(PROTOC_GEN_VALIDATE_GIT_URI).git deps/$(PROTOC_GEN_VALIDATE_GIT_URI)
	-git clone https://$(GOOGLEAPIS_GIT_URI).git deps/$(GOOGLEAPIS_GIT_URI)
	-git clone https://$(PROTOBUF_GIT_URI).git deps/$(PROTOBUF_GIT_URI)
	-git clone https://$(UDPA_GIT_URI).git deps/$(UDPA_GIT_URI)
###

proto: $(EXT_AUTH)external_auth.pb.go $(ADDRESS).pb.go $(V2_BASE).pb.go $(HTTP_STATUS).pb.go $(EXT_AUTH)attribute_context.pb.go proto/felixbackend.pb.go

$(EXT_AUTH)external_auth.pb.go $(EXT_AUTH)attribute_context.pb.go: $(EXT_AUTH)external_auth.proto $(EXT_AUTH)attribute_context.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(EXT_AUTH)*.proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(ADDRESS).pb.go $(V2_BASE).pb.go: $(ADDRESS).proto $(V2_BASE).proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(ADDRESS).proto $(V2_BASE).proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(HTTP_STATUS).pb.go: $(HTTP_STATUS).proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(HTTP_STATUS).proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(EXT_AUTH)external_auth.proto $(ADDRESS).proto $(V2_BASE).proto $(HTTP_STATUS).proto $(EXT_AUTH)attribute_context.proto:

proto/felixbackend.pb.go: proto/felixbackend.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              proto/*.proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):proto

###############################################################################
# Building the image
###############################################################################
.PHONY: image $(BUILD_IMAGES)

image-all: $(addprefix sub-image-,$(VALIDARCHES)) $(ENVOY_INIT_IMAGE)
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(L7_COLLECTOR_IMAGE)-$(ARCH)

$(L7_COLLECTOR_IMAGE)-$(ARCH): Dockerfile.$(ARCH) bin/l7-collector-$(ARCH)
	docker build -t $(L7_COLLECTOR_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(L7_COLLECTOR_IMAGE):latest-$(ARCH) $(L7_COLLECTOR_IMAGE):latest
endif

$(ENVOY_INIT_IMAGE):
	docker build -t $(ENVOY_INIT_IMAGE):latest-$(ARCH) -f envoy-init/Dockerfile.$(ARCH) envoy-init/.
ifeq ($(ARCH),amd64)
	docker tag $(ENVOY_INIT_IMAGE):latest-$(ARCH) $(ENVOY_INIT_IMAGE):latest
endif


###############################################################################
# Managing the upstream library pins
###############################################################################

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

## Update dependency pin
update-pins: guard-ssh-forwarding-bug replace-libcalico-pin

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
ut: proto bin/l7-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo -r --skipPackage deps,fv -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: fv
fv: proto bin/l7-collector-$(ARCH)
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