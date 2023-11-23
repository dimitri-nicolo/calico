PACKAGE_NAME?=github.com/tigera/elasticsearch-metrics
GO_BUILD_VER?=v0.90

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

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
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
			go build -buildvcs=false -o $@ -v -tags $(TAGS) -ldflags "$(LDFLAGS) -linkmode external -extldflags -static -s -w" cmd/*.go'
ifeq ($(ARCH), $(filter $(ARCH),amd64))
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'strings bin/elasticsearch-metrics-$(ARCH) | grep '_Cfunc__goboringcrypto_' 1> /dev/null'
endif

image: $(ELASTICSEARCH_METRICS_IMAGE)
$(ELASTICSEARCH_METRICS_IMAGE): $(ELASTICSEARCH_METRICS_IMAGE)-$(ARCH)
$(ELASTICSEARCH_METRICS_IMAGE)-$(ARCH): build
	docker buildx build --pull -t $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ELASTICSEARCH_METRICS_IMAGE):latest-$(ARCH) $(ELASTICSEARCH_METRICS_IMAGE):latest
endif

cd: image cd-common
