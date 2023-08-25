PACKAGE_NAME?=github.com/tigera/elasticsearch-docker
GO_BUILD_VER?=v0.87

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ELASTICSEARCH_DOCKER_PROJECT_ID)

ELASTICSEARCH_IMAGE   ?=tigera/elasticsearch
BUILD_IMAGES          ?=$(ELASTICSEARCH_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

ELASTIC_VERSION=7.17.11
GRADLE_VERSION=7.5.1
TINI_VERSION=0.19.0

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

ELASTICSEARCH_CONTAINER_MARKER=.elasticsearch_container-$(ARCH).created
ELASTICSEARCH_CONTAINER_FIPS_MARKER=.elasticsearch_container-$(ARCH)-fips.created

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common


# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

build: bin/readiness-probe-$(ARCH)

DOCKER_GO_BUILD_CGO=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e CGO_LDFLAGS=$(CGO_LDFLAGS) $(CALICO_BUILD)

.PHONY: bin/readiness-probe-$(ARCH)
bin/readiness-probe-$(ARCH): readiness-probe/main.go
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -ldflags="-s -w" -o $@ readiness-probe/main.go'

image: $(ELASTICSEARCH_IMAGE)

$(ELASTICSEARCH_IMAGE): $(ELASTICSEARCH_CONTAINER_MARKER) $(ELASTICSEARCH_CONTAINER_FIPS_MARKER)

$(ELASTICSEARCH_CONTAINER_MARKER): Dockerfile.$(ARCH) build
	docker buildx build --pull --load \
		--build-arg ELASTIC_VERSION=$(ELASTIC_VERSION) \
		--build-arg GRADLE_VERSION=$(GRADLE_VERSION) \
		--build-arg TINI_VERSION=$(TINI_VERSION) \
		-t $(ELASTICSEARCH_IMAGE):latest-$(ARCH) \
		-f Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

# build fips image
$(ELASTICSEARCH_CONTAINER_FIPS_MARKER): Dockerfile-fips.$(ARCH) build
	docker buildx build --pull --load \
		--build-arg ELASTIC_VERSION=$(ELASTIC_VERSION) \
		--build-arg GRADLE_VERSION=$(GRADLE_VERSION) \
		--build-arg TINI_VERSION=$(TINI_VERSION) \
		-t $(ELASTICSEARCH_IMAGE):latest-fips-$(ARCH) \
		-f Dockerfile-fips.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest-fips LATEST_IMAGE_TAG=latest-fips
	touch $@


.PHONY: cd
cd: image cd-common
	$(MAKE) FIPS=true retag-build-images-with-registries push-images-to-registries push-manifests IMAGETAG=$(if $(IMAGETAG_PREFIX),$(IMAGETAG_PREFIX)-)$(BRANCH_NAME)-fips LATEST_IMAGE_TAG=latest-fips
	$(MAKE) FIPS=true retag-build-images-with-registries push-images-to-registries push-manifests IMAGETAG=$(if $(IMAGETAG_PREFIX),$(IMAGETAG_PREFIX)-)$(shell git describe --tags --dirty --long --always --abbrev=12)-fips EXCLUDEARCH="$(EXCLUDEARCH)" LATEST_IMAGE_TAG=latest-fips

.PHONY: clean
clean:
	rm -rf bin .go-pkg-cache Makefile.*
	rm -f $(ELASTICSEARCH_CONTAINER_MARKER) $(ELASTICSEARCH_CONTAINER_FIPS_MARKER)
	-docker image rm -f $$(docker images $(ELASTICSEARCH_IMAGE) -a -q)
