include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/third_party/envoybinary

ENVOYBINARY_IMAGE ?= envoybinary
ENVOY_PROXY_IMAGE ?= envoy-proxy
BUILD_IMAGES ?= $(ENVOYBINARY_IMAGE) $(ENVOY_PROXY_IMAGE)

ENVOY_VERSION=v1.31.5
# https://github.com/envoyproxy/envoy/issues/2181
ENVOY_COMMIT=688c4bbe47f4d05bb8ed268f5172bb026cf03242

DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = $(ENVOY_VERSION)
LATEST_IMAGE_TAG = $(BRANCH_NAME)

##############################################################################
# Include lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../../lib.Makefile

##############################################################################
# Build
##############################################################################
ENVOY_DOWNLOADED=.envoy.downloaded

.PHONY: init-source
init-source:: $(ENVOY_DOWNLOADED)
$(ENVOY_DOWNLOADED):
	mkdir -p envoy
	curl -sfL https://github.com/envoyproxy/envoy/archive/refs/tags/${ENVOY_VERSION}.tar.gz | tar xz --strip-components 1 -C envoy
	patch -d envoy -p1 < patches/0001-use-localaddr-if-sock-is-transparent.patch
	touch $@

build: init-source
	cd envoy && \
	echo -n $(ENVOY_COMMIT) > SOURCE_VERSION && \
	ENVOY_DOCKER_BUILD_DIR=$(shell pwd)/envoy/build ci/run_envoy_docker.sh "ci/do_ci.sh release.server_only"

clean:
	rm -f ${ENVOY_DOWNLOADED} ${ENVOYBINARY_IMAGE_CREATED} ${ENVOY_PROXY_IMAGE_CREATED} Makefile.common*
	rm -fr envoy/
	-docker image rm -f $$(docker images $(ENVOYBINARY_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(ENVOY_PROXY_IMAGE) -a -q)

##############################################################################
# Image
##############################################################################
ENVOYBINARY_IMAGE_CREATED=.envoybinary.created-$(ARCH)
ENVOY_PROXY_IMAGE_CREATED=.envoy-proxy.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

# intermediate multi-arch image holds our patched envoy binary.
.PHONY: image
image: $(BUILD_IMAGES)

ENVOY_ARCH=$(ARCH)
ifeq ($(ARCH),amd64)
ENVOY_ARCH=x64
endif

$(ENVOYBINARY_IMAGE): $(ENVOYBINARY_IMAGE_CREATED)
$(ENVOYBINARY_IMAGE_CREATED): Dockerfile build
	$(DOCKER_BUILD) --target binary --build-arg ENVOY_ARCH=$(ENVOY_ARCH) \
		-t $(ENVOYBINARY_IMAGE):$(LATEST_IMAGE_TAG)-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=$(LATEST_IMAGE_TAG)
	touch $@

$(ENVOY_PROXY_IMAGE): $(ENVOY_PROXY_IMAGE_CREATED)
$(ENVOY_PROXY_IMAGE_CREATED): Dockerfile build
	$(DOCKER_BUILD) --target proxy --build-arg ENVOY_ARCH=$(ENVOY_ARCH) \
		-t $(ENVOY_PROXY_IMAGE):$(LATEST_IMAGE_TAG)-$(ARCH) -f Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=$(LATEST_IMAGE_TAG)
	touch $@

##############################################################################
# CI/CD
##############################################################################
.PHONY: cd
cd: image-all var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME
	$(MAKE) retag-build-images-with-registries push-images-to-registries IMAGETAG=$(LATEST_IMAGE_TAG) EXCLUDEARCH="$(EXCLUDEARCH)"

.PHONY: push-manifest
push-manifest:
	$(MAKE) push-manifests IMAGETAG=$(LATEST_IMAGE_TAG)
