.PHONY: ci cd
PACKAGE_NAME?=github.com/tigera/elasticsearch-docker
GO_BUILD_VER?=v0.50

GIT_VERSION?=$(shell git describe --tags --dirty --always --long  --abbrev=12)

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

BUILD_IMAGE_NAME=tigera/elasticsearch
BUILD_IMAGE_TAG=latest
BUILD_IMAGE=$(BUILD_IMAGE_NAME):$(BUILD_IMAGE_TAG)

PUSH_IMAGE_NAME?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE_NAME)

bin/readiness-probe: readiness-probe
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -o $@ "$(PACKAGE_NAME)/readiness-probe"';

build: bin/readiness-probe

image: build
	docker build --pull -t $(BUILD_IMAGE) .

compressed-image: image
	$(MAKE) docker-compress IMAGE_NAME=$(BUILD_IMAGE)

cd: compressed-image
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker tag $(BUILD_IMAGE) $(PUSH_IMAGE_NAME):$(BRANCH_NAME)
	docker tag $(BUILD_IMAGE) $(PUSH_IMAGE_NAME):$(GIT_VERSION)
	docker push $(PUSH_IMAGE_NAME):$(BRANCH_NAME)
	docker push $(PUSH_IMAGE_NAME):$(GIT_VERSION)

.PHONY: clean
clean:
	rm -rf bin \
		   .go-pkg-cache \
		   Makefile.*
