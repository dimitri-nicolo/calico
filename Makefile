PACKAGE_NAME?=github.com/tigera/egress-gateway
GO_BUILD_VER?=v0.53
GIT_USE_SSH=true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_EGRESS_GATEWAY_PROJECT_ID)

EGRESS_GATEWAY_IMAGE  ?=tigera/egress-gateway
BUILD_IMAGES          ?=$(EGRESS_GATEWAY_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

EXTRA_DOCKER_ARGS+=-e GOPRIVATE='github.com/tigera/*'

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

# Variables controlling the image
GATEWAY_CONTAINER_CREATED=.egress_gateway.created-$(ARCH)

# Files that go into the image
GATEWAY_CONTAINER_FILES=$(shell find ./filesystem -type f)

## Clean enough that a new release build will be clean
clean:
	find . -name '*.created' -exec rm -f {} +
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf .go-pkg-cache
	rm -rf certs *.tar
	rm -rf dist
	rm -rf Makefile.common*
	# Delete images that we built in this repo
	docker rmi $(EGRESS_GATEWAY_IMAGE):latest-$(ARCH) || true

###############################################################################
# Building the image
###############################################################################
## Create the image for the current ARCH
image: $(EGRESS_GATEWAY_IMAGE)
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(EGRESS_GATEWAY_IMAGE): $(GATEWAY_CONTAINER_CREATED)
$(GATEWAY_CONTAINER_CREATED): register ./Dockerfile.$(ARCH) $(GATEWAY_CONTAINER_FILES)
	docker build --pull -t $(EGRESS_GATEWAY_IMAGE):latest-$(ARCH) . --build-arg GIT_VERSION=$(GIT_VERSION) -f ./Dockerfile.$(ARCH)
	touch $@

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean image-all

## Deploys images to registry
cd: cd-common

$(info "Calico git version")
$(info $(shell printf "%-21s = %-10s\n" "GIT_VERSION" $(GIT_VERSION)))
