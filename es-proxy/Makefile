include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/es-proxy

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/kibana/Client \

TESLA ?= false

ES_PROXY_IMAGE        ?=es-proxy
BUILD_IMAGES          ?=$(ES_PROXY_IMAGE)

ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla/tigera
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-cloud
	DEV_TAG_SUFFIX        = cloud-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

BINDIR       ?= bin
BUILD_DIR    ?= build
TOP_SRC_DIRS  = pkg
SRC_FILES     = $(shell find pkg cmd -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../compliance/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')
TEST_DIRS    ?= $(shell find $(TOP_SRC_DIRS) -name '*_test.go' -exec dirname {} \; | sort | uniq)
ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/handler.VERSION=$(GIT_VERSION) \
	-X $(PACKAGE_NAME)/pkg/handler.BUILD_DATE=$(DATE) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_TAG=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_COMMIT=$(GIT_COMMIT) \
	-X main.VERSION=$(ES_PROXY_VERSION)

###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "es-proxy" instead of "$(BINDIR)/es-proxy".
build: $(BINDIR)/es-proxy-$(ARCH)

$(BINDIR)/es-proxy-$(ARCH): $(SRC_FILES)
ifeq ($(ARCH), amd64)
	$(DOCKER_RUN) -e CGO_ENABLED=1 $(CALICO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) \
	           go build -o $@ -v $(BUILD_TAGS) -ldflags "$(VERSION_FLAGS)" cmd/server/*.go'
else
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(BUILD_TAGS) -ldflags "$(VERSION_FLAGS)" cmd/server/*.go'
endif

ES_PROXY_CONTAINER_CREATED=.es-proxy.created-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(ES_PROXY_IMAGE)

$(ES_PROXY_IMAGE): $(ES_PROXY_CONTAINER_CREATED)
$(ES_PROXY_CONTAINER_CREATED): Dockerfile.$(ARCH) $(BINDIR)/es-proxy-$(ARCH)
	$(DOCKER_BUILD) -t $(ES_PROXY_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -fr $(BINDIR)
	rm -f $(ES_PROXY_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(ES_PROXY_IMAGE) -a -q)

##########################################################################
# Testing
##########################################################################
.PHONY: ut
ut:
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv: image run-k8s-apiserver run-elastic
	APISERVER_NAME=$(APISERVER_NAME) \
	PACKAGE_NAME=${PACKAGE_NAME} \
		test/setup_k8s_auth.sh

	PACKAGE_NAME=${PACKAGE_NAME} \
		test/setup_elasticsearch_index.sh

	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) ginkgo -r test/'

###############################################################################
# Static checks
###############################################################################
# See .golangci.yml for golangci-lint config
# SA1019 are deprecation checks, we don't want to fail on those because it means a library update that deprecates something
# requires immediate removing of the deprecated functions.
LINT_ARGS += --exclude SA1019

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut fv

## Deploys images to registry
cd: image-all cd-common
