PACKAGE_NAME?=github.com/tigera/eck-operator-docker
GO_BUILD_VER?=v0.55

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

build:

image: $(ECK_OPERATOR_IMAGE)
$(ECK_OPERATOR_IMAGE): $(ECK_OPERATOR_IMAGE)-$(ARCH)
$(ECK_OPERATOR_IMAGE)-$(ARCH): build eck-builder-image
	docker build --build-arg BUILDER_VERSION=$(BUILDER_VERSION) -t $(ECK_OPERATOR_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ECK_OPERATOR_IMAGE):latest-$(ARCH) $(ECK_OPERATOR_IMAGE):latest
endif

# replace cloud-on-k8s/Dockerfile ubi-minimal version with $(UBI_VERSION)
# replace cloud-on-k8s/Dockerfile golang version with $(GO_VERSION)
# then create a builder image that forms the basis for the tigera/eck-operator image.
eck-builder-image:
	git submodule update --init --recursive
	bash -l -c "\
		cd cloud-on-k8s && \
		sed -i 's/ubi-minimal\:[[:digit:]].[[:digit:]]\+/ubi-minimal\:$(UBI_VERSION)/g' Dockerfile && \
		sed -i 's/golang\:[[:digit:]].[[:digit:]]\+.[[:digit:]]/golang\:$(GO_VERSION)/g' Dockerfile && \
		OPERATOR_IMAGE=tigera/eck-operator-builder:$(BUILDER_VERSION) make docker-build && \
		git co Dockerfile"

compressed-image: image
	$(MAKE) docker-compress IMAGE_NAME=$(ECK_OPERATOR_IMAGE):latest

.PHONY: cd
cd: compressed-image cd-common

.PHONY: clean
clean:
	rm -rf bin \
		   .go-pkg-cache \
		   Makefile.*
