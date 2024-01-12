PACKAGE_NAME?=github.com/tigera/elasticsearch-metrics
GO_BUILD_VER?=v0.90

ELASTICSEARCH_METRICS_IMAGE ?=tigera/elasticsearch-metrics
BUILD_IMAGES                ?=$(ELASTICSEARCH_METRICS_IMAGE)
DEV_REGISTRIES              ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES          ?=quay.io
RELEASE_BRANCH_PREFIX       ?=release-calient
DEV_TAG_SUFFIX              ?=calient-0.dev

ARCHES=amd64 arm64

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ELASTICSEARCH_METRICS_PROJECT_ID)

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
FIPS ?= false

ifeq ($(FIPS),true)
CGO_ENABLED=1
GOEXPERIMENT=boringcrypto
TAGS=osusergo,netgo
else
CGO_ENABLED=0
endif

build: bin/elasticsearch-metrics-$(ARCH)

.PHONY: bin/elasticsearch-metrics-$(ARCH)
bin/elasticsearch-metrics-$(ARCH):
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e GOEXPERIMENT=$(GOEXPERIMENT) $(CALICO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -buildvcs=false -o $@ -v -ldflags="$(LDFLAGS) -s -w" -tags=$(TAGS) cmd/*.go'
ifeq ($(FIPS),true)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'strings bin/elasticsearch-metrics-$(ARCH) | grep '_Cfunc__goboringcrypto_' 1> /dev/null'
endif

clean:
	rm -rf bin Makefile.common*
	rm -f $(ELASTICSEARCH_METRICS_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(ELASTICSEARCH_METRICS_IMAGE) -a -q)

###############################################################################
# Image
###############################################################################
ELASTICSEARCH_METRICS_IMAGE_CREATED=.es-metrics.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(ELASTICSEARCH_METRICS_IMAGE)

$(ELASTICSEARCH_METRICS_IMAGE): $(ELASTICSEARCH_METRICS_IMAGE_CREATED)
$(ELASTICSEARCH_METRICS_IMAGE_CREATED): Dockerfile bin/elasticsearch-metrics-$(ARCH)
	docker buildx build --load --platform=linux/$(ARCH) --pull -t $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) -f ./Dockerfile .
ifeq ($(ARCH),amd64)
	docker tag $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) $(ELASTICSEARCH_METRICS_IMAGE):latest
endif

###############################################################################
# CI/CD
###############################################################################
cd: image-all cd-common
