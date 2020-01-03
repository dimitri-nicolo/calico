# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut fv

###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst Dockerfile.%,%,$(wildcard Dockerfile.*))

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
        BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
        BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
        override ARCH=amd64
endif

# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

# list of arches *not* to build when doing *-all
#    until s390x works correctly
EXCLUDEARCH ?= s390x
VALIDARCHES = $(filter-out $(EXCLUDEARCH),$(ARCHES))

###############################################################################
GO_BUILD_VER?=v0.30
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
PROTOC_VER?=v0.1
PROTOC_CONTAINER?=calico/protoc:$(PROTOC_VER)-$(BUILDARCH)

#### temp changes from common Makefile #####
BUILD_OS ?= $(shell uname -s | tr A-Z a-z)

ifneq ($(GOPATH),)
    GOMOD_CACHE = $(shell echo $(GOPATH) | cut -d':' -f1)/pkg/mod
else
    # If gopath is empty, default to $(HOME)/go.
    GOMOD_CACHE = $(HOME)/go/pkg/mod
endif

EXTRA_DOCKER_ARGS += -e GO111MODULE=on -v $(GOMOD_CACHE):/go/pkg/mod:rw

GIT_CONFIG_SSH ?= git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
##### temp changes from common Makefile #####


# Get version from git - used for releases.
INGRESS_GIT_VERSION?=$(shell git describe --tags --dirty --always)
INGRESS_BUILD_DATE?=$(shell date -u +'%FT%T%z')
INGRESS_GIT_REVISION?=$(shell git rev-parse --short HEAD)
INGRESS_GIT_DESCRIPTION?=$(shell git describe --tags)

ifeq ($(LOCAL_BUILD),true)
	INGRESS_GIT_VERSION = $(shell git describe --tags --dirty --always)-dev-build
endif

VERSION_FLAGS=-X main.VERSION=$(INGRESS_GIT_VERSION) \
	-X main.BUILD_DATE=$(INGRESS_BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(INGRESS_GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(INGRESS_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
LOCAL_USER_ID:=$(shell id -u)
MY_GID:=$(shell id -g)

SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

############################################################################
BUILD_IMAGE?=gcr.io/unique-caldron-775/cnx/tigera/ingress-collector
PUSH_IMAGES?=$(BUILD_IMAGE)
RELEASE_IMAGES?=quay.io/tigera/ingress-collector
PACKAGE_NAME?=github.com/tigera/ingress-collector

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

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif


DOCKER_RUN := mkdir -p .go-pkg-cache bin $(GOMOD_CACHE) && \
                  docker run --rm \
                      --net=host \
                      $(EXTRA_DOCKER_ARGS) \
                      -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                      -e GOCACHE=/go-cache \
                      -e GOPATH=/go \
                      -e OS=$(BUILD_OS) \
                      -e GOOS=$(BUILD_OS) \
                      -e GOARCH=$(ARCH) \
                      -e GOFLAGS=$(GOFLAGS) \
                      -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
                      -v $(CURDIR)/.go-pkg-cache:/go-cache:rw \
                      -w /go/src/$(PACKAGE_NAME)

DOCKER_RUN_RO := mkdir -p .go-pkg-cache bin $(GOMOD_CACHE) && \
                  docker run --rm \
                      --net=host \
                      $(EXTRA_DOCKER_ARGS) \
                      -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                      -e GOCACHE=/go-cache \
                      -e GOPATH=/go \
                      -e OS=$(BUILD_OS) \
                      -e GOOS=$(BUILD_OS) \
                      -e GOARCH=$(ARCH) \
                      -e GOFLAGS=$(GOFLAGS) \
                      -v $(CURDIR):/go/src/$(PACKAGE_NAME):ro \
                      -v $(CURDIR)/report:/go/src/$(PACKAGE_NAME)/report:rw \
                      -v $(CURDIR)/.go-pkg-cache:/go-cache:rw \
                      -w /go/src/$(PACKAGE_NAME)

 # Pre-configured docker run command that runs as this user with the repo
 # checked out to /code, uses the --rm flag to avoid leaving the container
 # around afterwards.
DOCKER_RUN_RM:=docker run --rm \
               $(EXTRA_DOCKER_ARGS) \
               --user $(LOCAL_USER_ID):$(MY_GID) -v $(CURDIR):/code

ENVOY_API=deps/github.com/envoyproxy/data-plane-api
EXT_AUTH=$(ENVOY_API)/envoy/service/auth/v2alpha/
ADDRESS=$(ENVOY_API)/envoy/api/v2/core/address
V2_BASE=$(ENVOY_API)/envoy/api/v2/core/base
HTTP_STATUS=$(ENVOY_API)/envoy/type/http_status

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	rm -rf report/
	rm -rf bin proto/felixbackend.pb.go
	-docker rmi $(BUILD_IMAGE):latest-$(ARCH)
	-docker rmi $(BUILD_IMAGE):$(VERSION)-$(ARCH)
ifeq ($(ARCH),amd64)
	-docker rmi $(BUILD_IMAGE):latest
	-docker rmi $(BUILD_IMAGE):$(VERSION)
endif
###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/ingress-collector-,$(VALIDARCHES))

.PHONY: build
## Build the binary for the current architecture and platform
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
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) && \
		go build $(LDFLAGS) -v -o bin/ingress-collector-$(ARCH) \
	   	./cmd/ingress-collector'

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
	$(DOCKER_RUN_RM) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(EXT_AUTH)*.proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(ADDRESS).pb.go $(V2_BASE).pb.go: $(ADDRESS).proto $(V2_BASE).proto
	$(DOCKER_RUN_RM) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(ADDRESS).proto $(V2_BASE).proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(HTTP_STATUS).pb.go: $(HTTP_STATUS).proto
	$(DOCKER_RUN_RM) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              $(HTTP_STATUS).proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):$(ENVOY_API)

$(EXT_AUTH)external_auth.proto $(ADDRESS).proto $(V2_BASE).proto $(HTTP_STATUS).proto $(EXT_AUTH)attribute_context.proto:

proto/felixbackend.pb.go: proto/felixbackend.proto
	$(DOCKER_RUN_RM) -v $(CURDIR):/src:rw \
	              $(PROTOC_CONTAINER) \
	              $(PROTOC_IMPORTS) \
	              proto/*.proto \
	              --gogofast_out=plugins=grpc,$(PROTOC_MAPPINGS):proto

###############################################################################
# Building the image
###############################################################################
CONTAINER_CREATED=.ingress-collector.created-$(ARCH)
.PHONY: image $(BUILD_IMAGE)
image: $(BUILD_IMAGE)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(BUILD_IMAGE): $(CONTAINER_CREATED)
$(CONTAINER_CREATED): Dockerfile.$(ARCH) bin/ingress-collector-$(ARCH)
	docker build -t $(BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif
	touch $@

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))
sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*:$(IMAGETAG))-ARCH --target $(call unescapefs,$*:$(IMAGETAG))"

## push default amd64 arch where multi-arch manifest is not supported
push-non-manifests: imagetag $(addprefix sub-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))
sub-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker push $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of one arch
tag-images: imagetag $(addprefix sub-single-tag-images-arch-,$(call escapefs,$(PUSH_IMAGES))) $(addprefix sub-single-tag-images-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))

sub-single-tag-images-arch-%:
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(VALIDARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

###############################################################################
# Managing the upstream library pins
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-libcalico-pin

## deprecated target alias
update-libcalico: update-pins
	$(warning !! Update update-libcalico is deprecated, use update-pins !!)

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

## Guard to ensure LIBCALICO repo and branch are reachable
guard-git-libcalico:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Ensure the branch exists, or set LIBCALICO_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(LIBCALICO_VERSION))" = "" ]; then \
		echo "ERROR: LIBCALICO version could not be determined"; \
		exit 1; \
	fi;

PIN_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LIBCALICO_BRANCH?=$(PIN_BRANCH)
LIBCALICO_REPO?=github.com/tigera/libcalico-go-private

update-libcalico-pin: guard-ssh-forwarding-bug guard-git-libcalico
	$(call update_replace_pin,github.com/projectcalico/libcalico-go,$(LIBCALICO_REPO),$(LIBCALICO_BRANCH))

###############################################################################
# Static checks
###############################################################################
## Perform static checks on the code.
.PHONY: static-checks
static-checks: guard-ssh-forwarding-bug
	$(DOCKER_RUN) -v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/deps \
		-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/fv \
	   	$(CALICO_BUILD) \
	   	golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 --timeout 5m --skip-dirs "deps$\" --disable govet

.PHONY: fix
## Fix static checks
fix:
	goimports -w $(SRC_FILES)

foss-checks:
	@echo Running $@...
	$(DOCKER_RUN_RO) \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  $(CALICO_BUILD) /usr/local/bin/fossa

###############################################################################
# UTs
###############################################################################
WHAT?=.
GINKGO_FOCUS?=.*

.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev
ut: proto bin/ingress-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN_RO) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) \
	    sh -c "ginkgo -r --skipPackage deps,fv -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: fv
fv: proto bin/ingress-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN_RO) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) \
	    sh -c "ginkgo fv -r --skipPackage deps -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

###############################################################################
# CI
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean build-all static-checks ut

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: image-all
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=${BRANCH_NAME} EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish"
	@echo ""

## Produces a git tag for the release.
release-tag: release-prereqs release-notes
	git tag $(VERSION) -F release-notes-$(VERSION)
	@echo ""
	@echo "Now you can build the release:"
	@echo ""
	@echo "  make VERSION=$(VERSION) release-build"
	@echo ""

## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
# Check that the correct code is checked out.
ifneq ($(VERSION), $(INGRESS_GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(INGRESS_GIT_VERSION))
endif

	$(MAKE) image-all
	$(MAKE) tag-images-all IMAGETAG=$(VERSION)
	# Generate the `latest` images.
	$(MAKE) tag-images-all IMAGETAG=latest

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	if ! docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) /ingress-collector --version | grep '^$(VERSION)$$'; then \
	  echo "Reported version:" `docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) /ingress-collector --version` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	# Push images.
	$(MAKE) push-all push-manifests push-non-manifests IMAGETAG=$(VERSION)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "If this is the latest stable release, then run the following to push 'latest' images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish-latest"
	@echo ""

# WARNING: Only run this target if this release is the latest stable release. Do NOT
# run this target for alpha / beta / release candidate builds, or patches to earlier Calico versions.
## Pushes `latest` release images. WARNING: Only run this for latest stable releases.
release-publish-latest: release-prereqs
	$(MAKE) push-all push-manifests push-non-manifests IMAGETAG=latest

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: help
## Display this help text
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	@awk '/^[a-zA-Z\-\_0-9\/]+:/ {                                      \
		nb = sub( /^## /, "", helpMsg );                                \
		if(nb == 0) {                                                   \
			helpMsg = $$0;                                              \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                   \
		}                                                               \
		if (nb)                                                         \
			printf "\033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;  \
	}                                                                   \
	{ helpMsg = $$0 }'                                                  \
	width=20                                                            \
	$(MAKEFILE_LIST)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks
