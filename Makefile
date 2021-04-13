PACKAGE_NAME?=github.com/tigera/elasticsearch-metrics
GO_BUILD_VER?=v0.51

BUILD_IMAGE=tigera/elasticsearch-metrics
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)

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

JUST_WATCH_VERSION=1.1.0
JUST_WATCH_FILE_NAME=elasticsearch_exporter-$(JUST_WATCH_VERSION).linux-amd64
JUST_WATCH_DOWNLOAD_URL=https://github.com/justwatchcom/elasticsearch_exporter/releases/download/v$(JUST_WATCH_VERSION)/$(JUST_WATCH_FILE_NAME).tar.gz

clean:
	rm -rf build \
		   Makefile.common*

build/$(JUST_WATCH_FILE_NAME).tar.gz:
	mkdir -p build bin
	wget -O build/$(JUST_WATCH_FILE_NAME).tar.gz $(JUST_WATCH_DOWNLOAD_URL)
	tar -xvf build/$(JUST_WATCH_FILE_NAME).tar.gz -C build
	cp build/$(JUST_WATCH_FILE_NAME)/elasticsearch_exporter bin

build: build/$(JUST_WATCH_FILE_NAME).tar.gz

image: $(BUILD_IMAGE)
$(BUILD_IMAGE): $(BUILD_IMAGE)-$(ARCH)
$(BUILD_IMAGE)-$(ARCH): build
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif

cd: image cd-common
