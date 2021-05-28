.PHONY: cd image
PACKAGE_NAME?=github.com/tigera/kibana-docker
GO_BUILD_VER?=v0.53

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_KIBANA_DOCKER_PROJECT_ID)

KIBANA_IMAGE          ?=tigera/kibana
BUILD_IMAGES          ?=$(KIBANA_IMAGE)
ARCHES                ?=amd64
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

BUILD_IMAGE_NAME?=tigera/kibana
BUILD_IMAGE_TAG?=latest
BUILD_IMAGE?=$(BUILD_IMAGE_NAME):$(BUILD_IMAGE_TAG)

PUSH_IMAGE_NAME?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE_NAME)

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
	git submodule init
	git submodule update

clean:
	rm -rf docker/Dockerfile \
		   Makefile.*
	bash -l -c '\
		cd kibana && \
		nvm install && nvm use && \
		yarn kbn clean && yarn cache clean'

kibana-bootstrap:
	bash -l -c '\
		cd kibana && \
		nvm install && nvm use && \
		yarn kbn bootstrap'

kibana-image: kibana-bootstrap
	bash -l -c '\
		cd kibana && \
		nvm install && nvm use && \
		yarn build --docker --skip-docker-ubi --no-oss --release'

BUILD_IMAGE_NAME?=tigera/kibana
BUILD_IMAGE_TAG?=latest
BUILD_IMAGE?=$(BUILD_IMAGE_NAME):$(BUILD_IMAGE_TAG)

PUSH_IMAGE_NAME?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE_NAME)

KIBANA_VERSION=$(shell jq -r '.version' kibana/package.json)

image: $(KIBANA_IMAGE)
$(KIBANA_IMAGE):
	cd docker && KIBANA_VERSION=$(KIBANA_VERSION) bash Dockerfile-template.sh
	docker build --build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) docker/. -t $(BUILD_IMAGE)
	docker build --pull -t $(KIBANA_IMAGE):latest-$(ARCH) --build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) --file ./docker/Dockerfile docker
ifeq ($(ARCH),amd64)
	docker tag $(KIBANA_IMAGE):latest-$(ARCH) $(KIBANA_IMAGE):latest
endif

compressed-image: image
	$(MAKE) docker-compress IMAGE_NAME=$(BUILD_IMAGE)

cd: compressed-image cd-common