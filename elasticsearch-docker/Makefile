PACKAGE_NAME?=github.com/tigera/elasticsearch-docker
GO_BUILD_VER?=v0.91

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ELASTICSEARCH_DOCKER_PROJECT_ID)

ELASTICSEARCH_IMAGE   ?=tigera/elasticsearch
BUILD_IMAGES          ?=$(ELASTICSEARCH_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

ARCHES=amd64 arm64

ELASTIC_VERSION=7.17.18
TINI_VERSION=0.19.0

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

ELASTICSEARCH_CONTAINER_MARKER=.elasticsearch_container-$(ARCH).created

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

DOCKER_GO_BUILD_CGO=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e GOEXPERIMENT=$(GOEXPERIMENT) $(CALICO_BUILD)
ELASTIC_DOWNLOADED=.elastic.downloaded

.PHONY: build
build: bin/readiness-probe-$(ARCH) build-es

.PHONY: bin/readiness-probe-$(ARCH)
bin/readiness-probe-$(ARCH): readiness-probe/main.go
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -tags $(TAGS) -ldflags="-s -w" -o $@ readiness-probe/main.go'

.PHONY: init-elastic
init-elastic: $(ELASTIC_DOWNLOADED)
$(ELASTIC_DOWNLOADED):
	mkdir -p build
	curl -sfL https://github.com/elastic/elasticsearch/archive/refs/tags/v$(ELASTIC_VERSION).tar.gz | tar xz -C build/
	patch -d build/elasticsearch-$(ELASTIC_VERSION) -p1 < patches/elastic-7.17.x-Update-dependencies-to-reduce-CVEs.patch
	touch $@

GRADLE_TASK=:distribution:archives:linux-tar:assemble
ifeq ($(ARCH),amd64)
	override GRADLE_TASK=:distribution:archives:linux-tar:assemble
else ifeq ($(ARCH),arm64)
	override GRADLE_TASK=:distribution:archives:linux-aarch64-tar:assemble
endif

.PHONY: build-es
build-es: init-elastic
	build/elasticsearch-$(ELASTIC_VERSION)/gradlew $(GRADLE_TASK) \
		-p build/elasticsearch-$(ELASTIC_VERSION) \
		-Dbuild.snapshot=false \
		-Dlicense.key=x-pack/license-tools/src/test/resources/public.key
	find build/elasticsearch-$(ELASTIC_VERSION)/ -name "elasticsearch-$(ELASTIC_VERSION)-linux-*.tar.gz" -exec cp {} build/ \;

.PHONY: clean
clean:
	rm -rf bin build .go-pkg-cache Makefile.*
	rm -f $(ELASTIC_DOWNLOADED) $(ELASTICSEARCH_CONTAINER_MARKER)
	-docker image rm -f $$(docker images $(ELASTICSEARCH_IMAGE) -a -q)

###############################################################################
# Image
###############################################################################
QEMU_IMAGE ?= calico/qemu-user-static:latest

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(ELASTICSEARCH_IMAGE)

$(ELASTICSEARCH_IMAGE): $(ELASTICSEARCH_CONTAINER_MARKER)

ELASTIC_ARCH=
OPENJDK_ARCH=
ifeq ($(ARCH),amd64)
	override ELASTIC_ARCH=x86_64
	override OPENJDK_ARCH=x64
else ifeq ($(ARCH),arm64)
	override ELASTIC_ARCH=aarch64
	override OPENJDK_ARCH=aarch64
endif

$(ELASTICSEARCH_CONTAINER_MARKER): register Dockerfile build
	docker buildx build --load --platform=linux/$(ARCH) --pull \
		--build-arg ELASTIC_ARCH=$(ELASTIC_ARCH) \
		--build-arg ELASTIC_VERSION=$(ELASTIC_VERSION) \
		--build-arg QEMU_IMAGE=$(QEMU_IMAGE) \
		--build-arg TINI_VERSION=$(TINI_VERSION) \
		-t $(ELASTICSEARCH_IMAGE):latest-$(ARCH) \
		-f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

###############################################################################
# Image
###############################################################################
.PHONY: cd
cd: image-all cd-common
