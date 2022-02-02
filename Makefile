PACKAGE_NAME?=github.com/tigera/elasticsearch-metrics
GO_BUILD_VER?=v0.65

ELASTICSEARCH_METRICS_IMAGE ?=tigera/elasticsearch-metrics
BUILD_IMAGES                ?=$(ELASTICSEARCH_METRICS_IMAGE)
DEV_REGISTRIES              ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES          ?=quay.io
RELEASE_BRANCH_PREFIX       ?=release-calient
DEV_TAG_SUFFIX              ?=calient-0.dev

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

clean:
	rm -rf bin \
		   Makefile.common*

build:
	git submodule update --init --recursive
	mkdir -p bin
	$(DOCKER_GO_BUILD) bash -x  -c "cd elasticsearch-exporter && make common-build"
	cp elasticsearch-exporter/elasticsearch_exporter bin/

image: $(ELASTICSEARCH_METRICS_IMAGE)
$(ELASTICSEARCH_METRICS_IMAGE): $(ELASTICSEARCH_METRICS_IMAGE)-$(ARCH)
$(ELASTICSEARCH_METRICS_IMAGE)-$(ARCH): build
	docker build --pull -t $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) $(ELASTICSEARCH_METRICS_IMAGE):latest
endif

cd: image cd-common
