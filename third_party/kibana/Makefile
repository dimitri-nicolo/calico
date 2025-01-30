include ../../metadata.mk

PACKAGE_NAME?=github.com/projectcalico/calico/third_party/kibana

KIBANA_IMAGE?=kibana-ubi
BUILD_IMAGES?=$(KIBANA_IMAGE)

KIBANA_VERSION = 8.17.1

# required for ci/cd
DEV_REGISTRIES = $(THIRD_PARTY_REGISTRY)
BRANCH_NAME = v$(KIBANA_VERSION)
LATEST_IMAGE_TAG = $(BRANCH_NAME)

# Set GTM_INTEGRATION explicitly so that in case the defaults change, we will still not
# accidentally enable the integration
GTM_INTEGRATION?=disabled

ifeq ($(TESLA),true)
GTM_INTEGRATION=enabled
endif

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
	git clone --depth 1 --branch v$(KIBANA_VERSION) --single-branch https://github.com/elastic/kibana.git
	patch -d kibana -p1 < patches/0001-apply-tigera-customization-to-kibana.patch
	patch -d kibana -p1 < patches/0002-Reduce-all-platforms-to-only-linux-variant.patch
	patch -d kibana -p1 < patches/0003-support-ubi-arm64-builds.patch
	patch -d kibana -p1 < patches/0004-update-to-offical-node-js-version-20.15.1.patch
	cp -r plugins/tigera kibana/plugins/
	cp -r plugins/google_tag_manager kibana/plugins/
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
	yarn build \
		--docker-cross-compile --docker-images \
		--skip-docker-cloud --skip-docker-contexts --skip-docker-fips \
		--skip-docker-serverless --skip-docker-ubuntu --skip-docker-wolfi \
		--release

# Build Tigera plugin
.PHONY: tigera-plugin
tigera-plugin:
	cd kibana/plugins/tigera/ && \
	. $(NVM_DIR)/nvm.sh && nvm install && nvm use && \
	yarn kbn bootstrap && yarn build --kibana-version $(LATEST_IMAGE_TAG)


# Build Google Tag Manager plugin
.PHONY: gtm-plugin
ifeq ($(GTM_INTEGRATION), true)
gtm-plugin:
	cd kibana/plugins/google_tag_manager/ && \
	. $(NVM_DIR)/nvm.sh && nvm install && nvm use && \
	yarn kbn bootstrap && yarn build --kibana-version $(LATEST_IMAGE_TAG)
else
gtm-plugin:
	@echo "GTM Integration is disabled. Skipping GTM plugin build."
endif


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

$(KIBANA_IMAGE_MARKER): tigera-plugin gtm-plugin
	docker buildx build --load --platform=linux/$(ARCH) \
		--build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) \
		--build-arg KIBANA_VERSION=$(KIBANA_VERSION) \
		--build-arg LATEST_IMAGE_TAG=$(LATEST_IMAGE_TAG) \
		--build-arg QEMU_IMAGE=$(QEMU_IMAGE) \
		-t $(KIBANA_IMAGE):$(LATEST_IMAGE_TAG)-$(ARCH) \
		-f Dockerfile .
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
