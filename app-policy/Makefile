include ../metadata.mk

PACKAGE_NAME = github.com/projectcalico/calico/app-policy

###############################################################################

GENERATED_FILES=../felix/proto/felixbackend.pb.go proto/healthz.pb.go
SRC_FILES=$(shell find . -name '*.go' |grep -v vendor) $(GENERATED_FILES)

RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

# Name of the images.
# e.g., <registry>/<name>:<tag>
DIKASTES_IMAGE ?=dikastes
BUILD_IMAGES ?= $(DIKASTES_IMAGE)

LDFLAGS = -X main.VERSION=$(GIT_VERSION)

##############################################################################
# Download and include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

CONTAINER_CREATED=image.created-$(ARCH)
CONTAINER_FIPS_CREATED=image.created-$(ARCH)-fips

FIPS ?= false

ifeq ($(FIPS),true)
CONTAINER_MARKER=$(CONTAINER_FIPS_CREATED)
VALIDARCHES=amd64
BINDIR=bin/$(ARCH)-fips
else
CONTAINER_MARKER=$(CONTAINER_CREATED)
BINDIR=bin
endif

# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	rm -rf .go-pkg-cache report vendor bin
	-docker rmi $(DIKASTES_IMAGE):latest-$(ARCH)
	-docker rmi $(DIKASTES_IMAGE):$(VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
	-docker rmi $(DIKASTES_IMAGE):latest
	-docker rmi $(DIKASTES_IMAGE):latest-fips-amd64
	-docker rmi $(DIKASTES_IMAGE):$(VERSION)
endif

.PHONY: clean-generated
# Delete (checked-in) generated files. Intentionally not part of the main clean target since these files are
# checked in and deleting them makes the repo "dirty" as far as git is concerned.
clean-generated:
	rm -rf $(GENERATED_FILES)

###############################################################################
# Building the binary
###############################################################################

.PHONY: build-all
## Build the binaries for all architectures and platforms
$(VALIDARCHES):
	$(MAKE) build ARCH=$@

build-all: $(VALIDARCHES)

.PHONY: build
## Build the binary for the current architecture and platform
build: 
	$(MAKE) $(BINDIR)/dikastes-$(ARCH) ARCH=$(ARCH)
	$(MAKE) $(BINDIR)/healthz-$(ARCH) ARCH=$(ARCH)

$(BINDIR)/dikastes-amd64: ARCH=amd64
$(BINDIR)/dikastes-arm64: ARCH=arm64
$(BINDIR)/dikastes-%: protobuf $(SRC_FILES)
ifeq ($(FIPS),true)
	$(call build_cgo_boring_binary, ./cmd/dikastes, $@)
else
	$(call build_cgo_binary, ./cmd/dikastes, $@)
endif

$(BINDIR)/healthz-amd64: ARCH=amd64
$(BINDIR)/healthz-arm64: ARCH=arm64
$(BINDIR)/healthz-%: protobuf $(SRC_FILES)
ifeq ($(FIPS),true)
	$(call build_cgo_boring_binary, ./cmd/healthz, $@)
else
	$(call build_binary, ./cmd/healthz, $@)
endif


###############################################################################
# Protobufs, 
#
# 1. defer to felix's makefile for felixbackend stuff
# 2. build proto for healthz
###############################################################################

../felix/proto/felixbackend.pb.go: ../felix/proto/felixbackend.proto
	$(MAKE) --directory ../felix protobuf

# We use gogofast for protobuf compilation.  Regular gogo is incompatible with
# gRPC, since gRPC uses golang/protobuf for marshalling/unmarshalling in that
# case.  See https://github.com/gogo/protobuf/issues/386 for more details.
# Note that we cannot seem to use gogofaster because of incompatibility with
# Envoy's validation library.
# When importing, we must use gogo versions of google/protobuf and
# google/rpc (aka googleapis).
PROTOC_IMPORTS =  -I proto\
		  -I ./

protobuf: $(GENERATED_FILES)

proto/healthz.pb.go: proto/healthz.proto
	$(DOCKER_RUN) -v $(CURDIR):/src:rw --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		      $(PROTOC_CONTAINER) \
		      $(PROTOC_IMPORTS) \
		      proto/*.proto \
		      --gogofast_out=plugins=grpc:proto
	$(MAKE) fix


# Building the image
###############################################################################
.PHONY: image $(DIKASTES_IMAGE)
image: $(DIKASTES_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES)) sub-image-fips-amd64
sub-image-%:
	$(MAKE) image ARCH=$*
sub-image-fips-%:
	$(MAKE) image FIPS=true ARCH=$*

$(DIKASTES_IMAGE): $(CONTAINER_MARKER)
$(CONTAINER_CREATED): Dockerfile.$(ARCH) $(BINDIR)/dikastes-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(DIKASTES_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

$(CONTAINER_FIPS_CREATED): Dockerfile.$(ARCH) $(BINDIR)/dikastes-$(ARCH) $(BINDIR)/healthz-$(ARCH)
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(DIKASTES_IMAGE):latest-fips-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest-fips LATEST_IMAGE_TAG=latest-fips
	touch $@
###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev
ut: protobuf ut-exec

ut-exec:
	$(DOCKER_GO_BUILD_CGO) /bin/bash -c "go install gotest.tools/gotestsum@latest && gotestsum --junitfile report/tests.xml ./..."


###############################################################################
# CI
###############################################################################

.PHONY: ci
ci: mod-download build-all check-generated-files static-checks ut

## Check if generated files are out of date
.PHONY: check-generated-files
check-generated-files:
	$(MAKE) clean
	$(MAKE) protobuf
	if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated files are out of date."; \
	  git diff; \
	  false; \
	else \
	  echo "Generated files are up to date."; \
	fi

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: image-all cd-common
