.PHONY: cd image
PACKAGE_NAME?=github.com/tigera/kibana-docker
GO_BUILD_VER?=v0.81

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_KIBANA_DOCKER_PROJECT_ID)

KIBANA_IMAGE          ?=tigera/kibana
BUILD_IMAGES          ?=$(KIBANA_IMAGE)
ARCHES                ?=amd64
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

KIBANA_VERSION = 7.17.9

# Set GTM_INTEGRATION explicitly so that in case the defaults change, we will still not
# accidentally enable the integration
GTM_INTEGRATION?=disabled

ifeq ($(TESLA),true)
GTM_INTEGRATION=enabled
IMAGETAG_PREFIX?=tesla
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

###############################################################################
# Build
###############################################################################
KIBANA_DOWNLOADED=.kibana.downloaded

# Add --squash argument for CICD pipeline runs only to avoid setting "experimental",
# for Docker processes on personal machine.
# set `DOCKER_BUILD=--squash make image` to squash images locally.
ifdef CI
DOCKER_BUILD+= --squash
endif

.PHONY: init-source
init-source: $(KIBANA_DOWNLOADED)
$(KIBANA_DOWNLOADED):
	mkdir -p kibana
	curl -sfL https://github.com/elastic/kibana/archive/refs/tags/v$(KIBANA_VERSION).tar.gz | tar xz --strip-components 1 -C kibana
	patch -d kibana -p1 < patches/0001-Apply-Tigera-customizations-to-Kibana.patch
	patch -d kibana -p1 < patches/0002-Upgrade-transitive-dependency-http-cache-semantics.patch
	touch $@

.PHONY: build
build: $(KIBANA_DOWNLOADED)
	cd kibana && \
	. $(NVM_DIR)/nvm.sh && nvm install && nvm use && \
	BUILD_TS_REFS_CACHE_ENABLE=false yarn kbn bootstrap && \
	yarn build --docker-images --skip-docker-ubuntu --release

.PHONY: clean
clean:
	rm -fr kibana/
	rm -f $(KIBANA_DOWNLOADED)
	-docker image rm -f $$(docker images $(KIBANA_IMAGE) -a -q)
	-docker image rm -f $$(docker images docker.elastic.co/kibana/kibana-ubi8 -a -q)

###############################################################################
# Image
###############################################################################
.PHONY: image
image: $(KIBANA_IMAGE)
$(KIBANA_IMAGE):
	docker build $(DOCKER_BUILD) \
		--build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) \
		--build-arg KIBANA_VERSION=$(KIBANA_VERSION) \
		-f docker/Dockerfile.amd64 \
		-t $(KIBANA_IMAGE):latest-$(ARCH) docker/
ifeq ($(ARCH),amd64)
	docker tag $(KIBANA_IMAGE):latest-$(ARCH) $(KIBANA_IMAGE):latest
endif

###############################################################################
# CD
###############################################################################
cd: image cd-common
