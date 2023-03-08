include ../metadata.mk

PACKAGE_NAME ?= github.com/tigera/calico-private/honeypod-prototype

##############################################################################
# constants
##############################################################################
HONEYPOD_IMAGE						?=honeypod
HONEYPOD_IMAGE_LOCATION				?=./compromised_k8s_pod/ip_enumeration/build
HONEYPOD_EXP_SERVICE_IMAGE			?=honeypod-exp-service
HONEYPOD_EXP_SERVICE_HTML_LOCATION	?= ./compromised_k8s_pod/exposed_service_nginx/build/html
HONEYPOD_EXP_SERVICE_LOCATION 		?=./compromised_k8s_pod/exposed_service_nginx/build
BUILD_IMAGES 						?=$(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)

RELEASE_BRANCH_PREFIX				?=release-calient
DEV_TAG_SUFFIX						?=calient-0.dev

LDFLAGS:=-ldflags "\
		-X $(PACKAGE_NAME)/pkg/version.VERSION=$(PKG_VERSION) \
		-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(PKG_VERSION_BUILD_DATE) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(PKG_VERSION_GIT_DESCRIPTION) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(PKG_VERSION_REVISION) \
		-B 0x$(BUILD_ID)"

##############################################################################
# Include ../lib.Makefile before anything else
#	 Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#	 that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# BUILD IMAGE
###############################################################################
# Build the docker image.
.PHONY: image $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)
image: $(BUILD_IMAGES)

HONEYPOD_CONTAINER_CREATED=.honeypod.created-$(ARCH)
HONEYPOD_EXP_SERVICE_CREATED=.honeypod-exp-service.created-$(ARCH)

$(HONEYPOD_IMAGE): $(HONEYPOD_CONTAINER_CREATED)
$(HONEYPOD_CONTAINER_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_IMAGE):latest-$(ARCH) -f $(HONEYPOD_IMAGE_LOCATION)/Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(HONEYPOD_IMAGE)
	touch $@


$(HONEYPOD_EXP_SERVICE_IMAGE): $(HONEYPOD_EXP_SERVICE_CREATED)
$(HONEYPOD_EXP_SERVICE_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_EXP_SERVICE_IMAGE):latest-$(ARCH) --build-arg HTML_LOCATION=$(HONEYPOD_EXP_SERVICE_HTML_LOCATION) -f $(HONEYPOD_EXP_SERVICE_LOCATION)/Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(HONEYPOD_EXP_SERVICE_IMAGE)
	touch $@

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd ci

ci: image

## Deploys images to registry
cd: image-all cd-common
