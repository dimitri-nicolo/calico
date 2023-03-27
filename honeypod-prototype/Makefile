include ../metadata.mk

PACKAGE_NAME ?= github.com/tigera/calico-private/honeypod-prototype

##############################################################################
# constants
##############################################################################
HONEYPOD_IMAGE                      ?=honeypod
HONEYPOD_EXP_SERVICE_IMAGE          ?=honeypod-exp-service
BUILD_IMAGES                        ?=$(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)
ARCHES                              ?=amd64

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
.PHONY: build
build:

# Build the docker image.
.PHONY: image $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)
image: $(BUILD_IMAGES)

HONEYPOD_IMAGE_CREATED=.honeypod.created-$(ARCH)
HONEYPOD_EXP_SERVICE_IMAGE_CREATED=.honeypod-exp-service.created-$(ARCH)

$(HONEYPOD_IMAGE): $(HONEYPOD_IMAGE_CREATED)
$(HONEYPOD_IMAGE_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_IMAGE):latest-$(ARCH) -f compromised_k8s_pod/ip_enumeration/Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(HONEYPOD_IMAGE)
	touch $@


$(HONEYPOD_EXP_SERVICE_IMAGE): $(HONEYPOD_EXP_SERVICE_IMAGE_CREATED)
$(HONEYPOD_EXP_SERVICE_IMAGE_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_EXP_SERVICE_IMAGE):latest-$(ARCH) -f compromised_k8s_pod/exposed_service_nginx/Dockerfile.$(ARCH) compromised_k8s_pod/exposed_service_nginx
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(HONEYPOD_EXP_SERVICE_IMAGE)
	touch $@

###############################################################################
# Clean
###############################################################################
.PHONY: clean
clean:
	rm -f $(HONEYPOD_EXP_SERVICE_IMAGE_CREATED)
	rm -f $(HONEYPOD_IMAGE_CREATED)
	-docker image rm -f $$(docker images $(HONEYPOD_EXP_SERVICE_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(HONEYPOD_IMAGE) -a -q)

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd ci

ci: image

## Deploys images to registry
cd: image-all cd-common
