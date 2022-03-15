PACKAGE_NAME?=github.com/tigera/eck-operator-docker
GO_BUILD_VER?=v0.65

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ECK_OPERATOR_DOCKER_PROJECT_ID)

ECK_OPERATOR_IMAGE   ?=tigera/eck-operator
BUILD_IMAGES          ?=$(ECK_OPERATOR_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

GO_VERSION  ?=1.17.5
UBI_VERSION ?=8.5
BUILDER_VERSION ?=v0.0.1

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

# Add --squash argument for CICD pipeline runs only to avoid setting "experimental",
# for Docker processes on personal machine.
# DOCKER_SQUASH is defaulted to be empty but can be set `DOCKER_SQUASH=--squash make image` 
# to squash images locally.
ifdef CI
DOCKER_SQUASH=--squash
endif

build:

image: $(ECK_OPERATOR_IMAGE)
$(ECK_OPERATOR_IMAGE): $(ECK_OPERATOR_IMAGE)-$(ARCH)
$(ECK_OPERATOR_IMAGE)-$(ARCH): build eck-builder-image
	docker build --build-arg BUILDER_VERSION=$(BUILDER_VERSION) $(DOCKER_SQUASH) -t $(ECK_OPERATOR_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ECK_OPERATOR_IMAGE):latest-$(ARCH) $(ECK_OPERATOR_IMAGE):latest
endif


# Replace cloud-on-k8s/Dockerfile ubi-minimal version with $(UBI_VERSION)
# Replace cloud-on-k8s/Dockerfile golang version with $(GO_VERSION)
# Create a builder image that forms the basis for the tigera/eck-operator image.
# Checkout the Dockerfile to revert it to the original
eck-builder-image:
	git submodule update --init --recursive
	$(DOCKER_GO_BUILD) bash -x prepare-create-eck-builder-image.sh
	bash -x create-eck-builder-image.sh tigera/eck-operator-builder:$(BUILDER_VERSION) $(UBI_VERSION) $(GO_VERSION)

.PHONY: cd
cd: image cd-common

.PHONY: clean
clean:
	rm -rf bin \
		   .go-pkg-cache \
		   Makefile.*
