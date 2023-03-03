PACKAGE_NAME            ?= github.com/tigera/honeypod-prototype
GO_BUILD_VER            ?= v0.75
GOMOD_VENDOR             = false
GIT_USE_SSH              = true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_HONEYPOD_PROTOTYPE_PROJECT_ID)

ifdef GOPATH
EXTRA_DOCKER_ARGS += -v $(GOPATH)/pkg/mod:/go/pkg/mod:rw
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

##############################################################################
# constants
##############################################################################
HONEYPOD_IMAGE 								?=tigera/honeypod
HONEYPOD_IMAGE_LOCATION     	?=./compromised_k8s_pod/ip_enumeration/build
HONEYPOD_EXP_SERVICE_IMAGE 		?=tigera/honeypod-exp-service
HONEYPOD_EXP_SERVICE_HTML_LOCATION ?= ./compromised_k8s_pod/exposed_service_nginx/build/html
HONEYPOD_EXP_SERVICE_LOCATION ?=./compromised_k8s_pod/exposed_service_nginx/build
BUILD_IMAGES 									?=$(HONEYPOD_IMAGE) $(HONEYPOD_EXP_SERVICE_IMAGE)
ARCHES                    		?=amd64
DEV_REGISTRIES            		?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES        		?=quay.io
RELEASE_BRANCH_PREFIX     		?=release-calient
DEV_TAG_SUFFIX            		?=calient-0.dev

LDFLAGS:=-ldflags "\
		-X $(PACKAGE_NAME)/pkg/version.VERSION=$(PKG_VERSION) \
		-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(PKG_VERSION_BUILD_DATE) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(PKG_VERSION_GIT_DESCRIPTION) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(PKG_VERSION_REVISION) \
		-B 0x$(BUILD_ID)"

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

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

$(HONEYPOD_IMAGE): $(HONEYPOD_IMAGE)-$(ARCH)
$(HONEYPOD_IMAGE)-$(ARCH): 
	docker build -t $(HONEYPOD_IMAGE):latest-$(ARCH) --file $(HONEYPOD_IMAGE_LOCATION)/Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(HONEYPOD_IMAGE):latest-$(ARCH) $(HONEYPOD_IMAGE):latest
endif

$(HONEYPOD_EXP_SERVICE_IMAGE): $(HONEYPOD_EXP_SERVICE_IMAGE)-$(ARCH)
$(HONEYPOD_EXP_SERVICE_IMAGE)-$(ARCH): 
	docker build -t $(HONEYPOD_EXP_SERVICE_IMAGE):latest-$(ARCH) --build-arg HTML_LOCATION=$(HONEYPOD_EXP_SERVICE_HTML_LOCATION) --file $(HONEYPOD_EXP_SERVICE_LOCATION)/Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(HONEYPOD_EXP_SERVICE_IMAGE):latest-$(ARCH) $(HONEYPOD_EXP_SERVICE_IMAGE):latest
endif

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd ci

ci: image

## Deploys images to registry
cd: image-all cd-common
