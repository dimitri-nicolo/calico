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
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-tesla
	DEV_TAG_SUFFIX        = tesla-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

# Add in local static-checks
LOCAL_CHECKS=check-boring-ssl

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/elasticsearch/ClientBuilder \

SRC_FILES=cmd/kube-controllers/main.go $(shell find pkg -name '*.go') $(shell find ../libcalico-go -name '*.go')

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
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
	-docker rmi $(FLANNEL_MIGRATION_IMAGE)
	-docker rmi $(FLANNEL_MIGRATION_IMAGE):latest-amd64
	rm -f tests/fv/fv.test
	rm -f report/*.xml

###############################################################################
# Building the binary
###############################################################################
build: bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH)
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/kube-controllers-linux-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) \
	  -e CGO_ENABLED=$(CGO_ENABLED) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS) -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/kube-controllers/'

bin/wrapper-$(ARCH):
	$(DOCKER_RUN) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS)  -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/wrapper'

bin/check-status-linux-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) \
	  -e CGO_ENABLED=$(CGO_ENABLED) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS)  -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/check-status/'

bin/kubectl-$(ARCH):
	wget https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VERSION)/bin/linux/$(subst armv7,arm,$(ARCH))/kubectl -O $@
	chmod +x $@

###############################################################################
# Building the image
###############################################################################
## Builds the controller binary and docker image.
image: image.created-$(ARCH)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image.created-$(ARCH): bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH) bin/wrapper-$(ARCH) bin/kubectl-$(ARCH)
	$(DOCKER_BUILD) -t $(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(DOCKER_BUILD) -t $(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) -f docker-images/flannel-migration/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

###############################################################################
# Static checks
###############################################################################
# Make sure that a copyright statement exists on all go files.
check-copyright:
	./check-copyrights.sh

check-boring-ssl: bin/kube-controllers-linux-amd64
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD) \
		go tool nm bin/kube-controllers-linux-amd64 > bin/tags.txt && grep '_Cfunc__goboringcrypto_' bin/tags.txt 1> /dev/null
	-rm -f bin/tags.txt

###############################################################################
# Tests
###############################################################################
ifeq ($(SEMAPHORE_GIT_REF_TYPE), pull-request)
# Determine the tests to run using the test spider tool, which emits a list of impacted packages.
WHAT=$(shell $(DOCKER_GO_BUILD) $(GIT_CONFIG_SSH) go run ../hack/test/spider -commit-range=${SEMAPHORE_GIT_COMMIT_RANGE} -filter-dir kube-controllers/)
else
# By default, run all tests.
WHAT=$(shell find . -name "*_test.go" | xargs dirname | sort -u)
endif

# The list of test binaries to build.
# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
TEST_BINARIES=$(addsuffix /ut.test,$(WHAT))

## Run the unit tests in a container.
ut fv: $(TEST_BINARIES)
	KUBE_IMAGE=$(CALICO_BUILD) \
		   ETCD_IMAGE=$(ETCD_IMAGE) \
		   CONTAINER_NAME=$(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) \
		   MIGRATION_CONTAINER_NAME=$(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) \
		   CRDS=$(CURDIR)/../libcalico-go/config/crd \
		   CERTS_PATH=$(CERTS_PATH) \
		   ./run-uts $(WHAT)

# Produce test binaries for each package that needs them.
# ginkgo doesn't let you produce a single test binary with multiple packages.
%/ut.test: $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./$* -c --tags fvtests -o $@'

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
