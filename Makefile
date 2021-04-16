.PHONY: cd image
PACKAGE_NAME?=github.com/tigera/kibana-docker
GO_BUILD_VER?=v0.51

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_KIBANA_DOCKER_PROJECT_ID)

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

image:
	cd docker && KIBANA_VERSION=$(KIBANA_VERSION) bash Dockerfile-template.sh
	docker build --build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) docker/. -t $(BUILD_IMAGE)

compressed-image: image
	$(MAKE) docker-compress IMAGE_NAME=$(BUILD_IMAGE)

# Set the image tags if the branch name is defined
ifdef BRANCH_NAME
ifdef IMAGE_PREFIX
BRANCH_NAME_TAG=$(IMAGE_PREFIX)-$(BRANCH_NAME)
GIT_VERSION_TAG=$(IMAGE_PREFIX)-$(GIT_VERSION)
else
BRANCH_NAME_TAG=$(BRANCH_NAME)
GIT_VERSION_TAG=$(GIT_VERSION)
endif
endif

tag-kibana-images:
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker tag $(BUILD_IMAGE) $(PUSH_IMAGE_NAME):$(BRANCH_NAME_TAG)
	docker tag $(BUILD_IMAGE) $(PUSH_IMAGE_NAME):$(GIT_VERSION_TAG)

cd: compressed-image tag-kibana-images
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
	docker push $(PUSH_IMAGE_NAME):$(BRANCH_NAME_TAG)
	docker push $(PUSH_IMAGE_NAME):$(GIT_VERSION_TAG)
