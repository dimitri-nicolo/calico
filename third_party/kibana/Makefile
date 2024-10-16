include ../../metadata.mk

PACKAGE_NAME?=github.com/projectcalico/calico/third_party/kibana

KIBANA_IMAGE?=kibana-ubi
BUILD_IMAGES?=$(KIBANA_IMAGE)

KIBANA_VERSION = 7.17.24

# required for ci/cd
DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = v$(KIBANA_VERSION)
LATEST_IMAGE_TAG = $(BRANCH_NAME)

###############################################################################
# Include lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../../lib.Makefile

###############################################################################
# Build
###############################################################################
KIBANA_DOWNLOADED=.kibana.downloaded

.PHONY: init-source
init-source: $(KIBANA_DOWNLOADED)
$(KIBANA_DOWNLOADED):
	mkdir -p kibana
	curl -sfL https://github.com/elastic/kibana/archive/refs/tags/v$(KIBANA_VERSION).tar.gz | tar xz --strip-components 1 -C kibana
	patch -d kibana -p1 < patches/0001-Apply-Tigera-customizations-to-Kibana.patch
	patch -d kibana -p1 < patches/0002-Reduce-all-platforms-to-linux-variants-only.patch
	patch -d kibana -p1 < patches/0003-Support-UBI-arm64-builds.patch
	patch -d kibana -p1 < patches/0004-Update-packages.patch
	touch $@

# always register because build target will build images for all supported arches
.PHONY: register
register:
	docker run --rm --privileged multiarch/qemu-user-static:register || true

.PHONY: build
build: register $(KIBANA_DOWNLOADED)
	cd kibana && \
	. $(NVM_DIR)/nvm.sh && nvm install && nvm use && \
	BUILD_TS_REFS_CACHE_ENABLE=false yarn kbn bootstrap && \
	yarn build --docker-images --docker-cross-compile --skip-docker-ubuntu --release

.PHONY: clean
clean:
	rm -fr kibana/
	rm -f $(KIBANA_DOWNLOADED)
	-docker image rm -f $$(docker images docker.elastic.co/kibana/kibana-ubi -a -q)

###############################################################################
# Image
###############################################################################
KIBANA_IMAGE_MARKER=.kibana.created-$(ARCH)

.PHONY: image-all
image-all: build $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(KIBANA_IMAGE)

$(KIBANA_IMAGE): $(KIBANA_IMAGE_MARKER)

$(KIBANA_IMAGE_MARKER):
	docker tag docker.elastic.co/kibana/$(KIBANA_IMAGE):$(KIBANA_VERSION)-$(ARCH) $(KIBANA_IMAGE):$(LATEST_IMAGE_TAG)-$(ARCH)
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=$(LATEST_IMAGE_TAG)
	touch $@

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd
cd: image-all var-require-one-of-CONFIRM-DRYRUN var-require-all-BRANCH_NAME
	$(MAKE) retag-build-images-with-registries push-images-to-registries IMAGETAG=$(LATEST_IMAGE_TAG) EXCLUDEARCH="$(EXCLUDEARCH)"

.PHONY: push-manifest
push-manifest:
	$(MAKE) push-manifests IMAGETAG=$(LATEST_IMAGE_TAG)
