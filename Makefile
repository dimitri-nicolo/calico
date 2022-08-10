PACKAGE_NAME?=github.com/tigera/elasticsearch-docker
GO_BUILD_VER?=v0.73

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ELASTICSEARCH_DOCKER_PROJECT_ID)

ELASTICSEARCH_IMAGE   ?=tigera/elasticsearch
BUILD_IMAGES          ?=$(ELASTICSEARCH_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# Add --squash argument for CICD pipeline runs only to avoid setting "experimental",
# for Docker processes on personal machine.
# set `DOCKER_BUILD=--squash make image` to squash images locally.
ifdef CI
DOCKER_BUILD+= --squash
endif

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

bin/readiness-probe: readiness-probe
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -o $@ "$(PACKAGE_NAME)/readiness-probe"';

build: bin/readiness-probe

image: $(ELASTICSEARCH_IMAGE)
$(ELASTICSEARCH_IMAGE): $(ELASTICSEARCH_IMAGE)-$(ARCH)
$(ELASTICSEARCH_IMAGE)-$(ARCH): build
	docker build $(DOCKER_BUILD) --pull -t $(ELASTICSEARCH_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ELASTICSEARCH_IMAGE):latest-$(ARCH) $(ELASTICSEARCH_IMAGE):latest
	# build fips image for amd64
	docker build $(DOCKER_BUILD) --pull -t $(ELASTICSEARCH_IMAGE):latest-amd64-fips --file ./Dockerfile.amd64-fips .
endif

.PHONY: cd
cd: image cd-common

.PHONY: clean
clean:
	rm -rf bin \
		   .go-pkg-cache \
		   Makefile.*
