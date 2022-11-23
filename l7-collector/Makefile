PACKAGE_NAME    ?= github.com/tigera/l7-collector
GO_BUILD_VER    ?= v0.76
GIT_USE_SSH     := true

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
ifdef CALICO_PATH
EXTRA_DOCKER_ARGS += -v $(CALICO_PATH):/go/src/github.com/projectcalico/calico/:ro
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

SRC_FILES=$(shell find . -name '*.go' |grep -v vendor)

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

# Set version info from git.
BUILD_DATE ?= $(shell date -u +'%FT%T%z')
GIT_REVISION ?= $(shell git rev-parse --short HEAD)

VERSION_FLAGS=-X main.VERSION=$(GIT_VERSION) \
	-X main.BUILD_DATE=$(BUILD_DATE) \
	-X main.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(GIT_REVISION)
BUILD_LDFLAGS = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

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
bin/l7-collector-%: protobuf $(SRC_FILES)
ifndef VERSION
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LD_FLAGS))
endif
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build $(LDFLAGS) -v -o bin/l7-collector-$(ARCH) \
	   	./cmd/l7-collector'

# Generate the protobuf bindings for go. The proto/felixbackend.pb.go file is included in SRC_FILES
# In order to make use of complex structures like protobuf.google.timestamp, we need to link the
# protobuf google types when generating go code from protobuf messages
protobuf proto/felixbackend.pb.go: proto/felixbackend.proto
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

update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))

## Update dependency pin
update-pins: guard-ssh-forwarding-bug update-calico-pin

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
ut: protobuf bin/l7-collector-$(ARCH)
	mkdir -p report
	$(DOCKER_RUN) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo -r --skipPackage deps,fv -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: fv
fv: protobuf bin/l7-collector-$(ARCH)
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
