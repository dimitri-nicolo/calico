include ../metadata.mk

PACKAGE_NAME ?= github.com/tigera/calico-private/honeypod-prototype

##############################################################################
# constants
##############################################################################
HONEYPOD_IMAGE                      ?=honeypod
HONEYPOD_EXP_SERVICE_IMAGE          ?=honeypod-exp-service
BUILD_IMAGES                        ?=$(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)

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
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)
image: $(BUILD_IMAGES)

HONEYPOD_IMAGE_CREATED=.honeypod.created-$(ARCH)
HONEYPOD_EXP_SERVICE_IMAGE_CREATED=.honeypod-exp-service.created-$(ARCH)

$(HONEYPOD_IMAGE): $(HONEYPOD_IMAGE_CREATED)
$(HONEYPOD_IMAGE_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_IMAGE):latest-$(ARCH) -f compromised_k8s_pod/ip_enumeration/Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(HONEYPOD_IMAGE)
	touch $@

$(HONEYPOD_EXP_SERVICE_IMAGE): $(HONEYPOD_EXP_SERVICE_IMAGE_CREATED)
$(HONEYPOD_EXP_SERVICE_IMAGE_CREATED):
	$(DOCKER_BUILD) -t $(HONEYPOD_EXP_SERVICE_IMAGE):latest-$(ARCH) -f compromised_k8s_pod/exposed_service_nginx/Dockerfile compromised_k8s_pod/exposed_service_nginx
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
