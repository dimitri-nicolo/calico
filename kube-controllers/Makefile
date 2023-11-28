include ../metadata.mk

PACKAGE_NAME = github.com/projectcalico/calico/kube-controllers
TESLA ?= false

# Name of the images.
# e.g., <registry>/<name>:<tag>
KUBE_CONTROLLERS_IMAGE  ?=kube-controllers
FLANNEL_MIGRATION_IMAGE ?=flannel-migration-controller

# Makefile configuration options
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

ifdef RELEASE
BUILD_IMAGES ?=$(KUBE_CONTROLLERS_IMAGE)
else
BUILD_IMAGES ?=$(KUBE_CONTROLLERS_IMAGE) $(FLANNEL_MIGRATION_IMAGE)
endif

ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla/tigera
	BUILD_TAGS            ?= tesla
	RELEASE_BRANCH_PREFIX = release-tesla
	DEV_TAG_SUFFIX        = tesla-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/elasticsearch/ClientBuilder \

SRC_FILES=$(shell find cmd -name '*.go') $(shell find pkg -name '*.go') $(shell find ../libcalico-go -name '*.go')
KUBE_CONTROLLER_CONTAINER_CREATED=image.created-$(ARCH)
KUBE_CONTROLLER_CONTAINER_FIPS_CREATED=image.created-$(ARCH)-fips

FIPS ?= false

ifeq ($(FIPS), true)
KUBE_CONTROLLER_CONTAINER_MARKER=$(KUBE_CONTROLLER_CONTAINER_FIPS_CREATED)
VALIDARCHES=amd64
BINDIR=bin/$(ARCH)-fips
else
KUBE_CONTROLLER_CONTAINER_MARKER=$(KUBE_CONTROLLER_CONTAINER_CREATED)
BINDIR=bin
endif

###############################################################################

## Removes all build artifacts.
clean:
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	rm -rf .go-pkg-cache bin image.created-$(ARCH) build report/*.xml release-notes-*
	-docker rmi $(KUBE_CONTROLLERS_IMAGE)
	-docker rmi $(KUBE_CONTROLLERS_IMAGE):latest-amd64
	-docker rmi $(KUBE_CONTROLLERS_IMAGE):latest-fips-amd64
	-docker rmi $(FLANNEL_MIGRATION_IMAGE)
	-docker rmi $(FLANNEL_MIGRATION_IMAGE):latest-amd64
	rm -f tests/fv/fv.test
	rm -f report/*.xml

###############################################################################
# Building the binary
###############################################################################
LDFLAGS = -X main.VERSION=$(GIT_VERSION)

build: $(BINDIR)/kube-controllers-linux-$(ARCH) $(BINDIR)/check-status-linux-$(ARCH)
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

$(BINDIR)/kube-controllers-linux-$(ARCH): $(SRC_FILES)
ifeq ($(ARCH), amd64)
ifeq ($(FIPS), true)
	$(call build_cgo_boring_binary, ./cmd/kube-controllers/, $@)
else
	$(call build_binary, ./cmd/kube-controllers/, $@)
endif
else
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '\
		$(GIT_CONFIG_SSH) go build $(if $(BUILD_TAGS),-tags $(BUILD_TAGS)) -buildvcs=false -v -o $@ -ldflags "$(LDFLAGS) -s -w" ./cmd/kube-controllers/'
endif

$(BINDIR)/wrapper-$(ARCH):
ifeq ($(ARCH), amd64)
ifeq ($(FIPS), true)
	$(call build_cgo_boring_binary, ./cmd/wrapper, $@)
else
	$(call build_binary, ./cmd/wrapper, $@)
endif
else
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '\
		$(GIT_CONFIG_SSH) go build $(if $(BUILD_TAGS),-tags $(BUILD_TAGS)) -buildvcs=false -v -o ./cmd/kube-controllers/ -ldflags "$(LDFLAGS) -s -w" ./cmd/wrapper'
endif

$(BINDIR)/check-status-linux-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) \
	  -e CGO_ENABLED=$(CGO_ENABLED) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(if $(BUILD_TAGS),-tags $(BUILD_TAGS)) -buildvcs=false -v -o $@ -ldflags "$(LDFLAGS) -s -w" ./cmd/check-status/'

$(BINDIR)/kubectl-$(ARCH):
	curl -sSf -L --retry 5 -o $@ https://storage.googleapis.com/kubernetes-release/release/$(K8S_VERSION)/bin/linux/$(ARCH)/kubectl
	chmod +x $@

###############################################################################
# Building the image
###############################################################################
## Builds the controller binary and docker image.
image-all: $(addprefix sub-image-,$(VALIDARCHES)) sub-image-fips-amd64
sub-image-%:
	$(MAKE) image ARCH=$*
sub-image-fips-%:
	$(MAKE) image FIPS=true ARCH=$*

image: $(KUBE_CONTROLLER_CONTAINER_MARKER)

$(KUBE_CONTROLLER_CONTAINER_CREATED): $(BINDIR)/kube-controllers-linux-$(ARCH) $(BINDIR)/check-status-linux-$(ARCH) $(BINDIR)/wrapper-$(ARCH) $(BINDIR)/kubectl-$(ARCH) register
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) -f docker-images/flannel-migration/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

$(KUBE_CONTROLLER_CONTAINER_FIPS_CREATED): $(BINDIR)/kube-controllers-linux-$(ARCH) $(BINDIR)/check-status-linux-$(ARCH) $(BINDIR)/kubectl-$(ARCH) register
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(KUBE_CONTROLLERS_IMAGE):latest-fips-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(DOCKER_BUILD) --build-arg BIN_DIR=$(BINDIR) -t $(FLANNEL_MIGRATION_IMAGE):latest-fips-$(ARCH) -f docker-images/flannel-migration/Dockerfile.$(ARCH) . --load
	$(MAKE) FIPS=true retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest-fips LATEST_IMAGE_TAG=latest-fips
	touch $@

###############################################################################
# Static checks
###############################################################################
# Make sure that a copyright statement exists on all go files.
check-copyright:
	./check-copyrights.sh

###############################################################################
# Tests
###############################################################################
ifeq ($(SEMAPHORE_GIT_REF_TYPE), pull-request)
# Determine the tests to run using the test spider tool, which emits a list of impacted packages.
MAYBE_WHAT=$(shell $(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go run ../hack/test/spider -commit-range=${SEMAPHORE_GIT_COMMIT_RANGE} -filter-dir kube-controllers/')
else
# By default, run all tests.
MAYBE_WHAT=.
endif

# Filter the list of directories to only those that have tests.
ifneq ("$(MAYBE_WHAT)","")
WHAT=$(shell find $(MAYBE_WHAT) -name "*_test.go" | xargs dirname | sort -u)
endif

# The list of test binaries to build.
# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
TEST_BINARIES=$(addsuffix /ut.test,$(WHAT))

## Run the unit tests in a container.
test: ut
ut fv: $(TEST_BINARIES) image
	KUBE_IMAGE=$(CALICO_BUILD) \
		   ETCD_IMAGE=$(ETCD_IMAGE) \
		   CONTAINER_NAME=$(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) \
		   MIGRATION_CONTAINER_NAME=$(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) \
		   CRDS=$(CURDIR)/../libcalico-go/config/crd \
		   CERTS_PATH=$(CERTS_PATH) \
		   ./run-uts $(WHAT)

st:
	@echo "No ST tests specified"

# Produce test binaries for each package that needs them.
# ginkgo doesn't let you produce a single test binary with multiple packages.
# Only do this if there are .go files in the path.
%/ut.test: $(SRC_FILES)
	if [ $$(find ./$* -name '*.go' | wc -l) -gt 0 ]; then \
		$(DOCKER_RUN) -e CGO_ENABLED=0 $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./$* -c --tags fvtests -o $@'; \
	else \
		echo "Skipping $* as it has no .go files in it"; \
	fi

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean mod-download image-all static-checks ut

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: cd-common
