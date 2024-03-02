include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/envoy

ENVOY_IMAGE ?= envoy
BUILD_IMAGES ?= $(ENVOY_IMAGE)

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

##############################################################################
# Build
##############################################################################
ENVOY_DOWNLOADED=.envoy.downloaded

ENVOY_VERSION=v1.29.0
# https://github.com/envoyproxy/envoy/issues/2181
ENVOY_COMMIT=a6d1d66a62b985baed414ba90ad0daebfc074664

.PHONY: init-source
init-source:: $(ENVOY_DOWNLOADED)
$(ENVOY_DOWNLOADED):
	mkdir -p envoy
	curl -sfL https://github.com/envoyproxy/envoy/archive/refs/tags/${ENVOY_VERSION}.tar.gz | tar xz --strip-components 1 -C envoy
	touch $@

build: init-source
	cd envoy && \
	echo -n $(ENVOY_COMMIT) > SOURCE_VERSION && \
	ENVOY_DOCKER_BUILD_DIR=$(shell pwd)/envoy/build ci/run_envoy_docker.sh "ci/do_ci.sh release.server_only"

clean:
	rm -f ${ENVOY_DOWNLOADED} ${ENVOY_IMAGE_CREATED} Makefile.common*
	rm -fr envoy/
	-docker image rm -f $$(docker images $(ENVOY_IMAGE) -a -q)

##############################################################################
# Image
##############################################################################
ENVOY_IMAGE_CREATED=.envoy.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(ENVOY_IMAGE)

# We don't have any custom fix to apply so build step is skipped
$(ENVOY_IMAGE): $(ENVOY_IMAGE_CREATED)
$(ENVOY_IMAGE_CREATED): register Dockerfile
	$(DOCKER_BUILD) --build-arg ENVOY_VERSION=$(ENVOY_VERSION) \
		-t $(ENVOY_IMAGE):latest-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

##############################################################################
# CI/CD
##############################################################################
.PHONY: ci
ci: image

.PHONY: cd
cd: image-all cd-common
