GO_BUILD_VER   ?= v0.53
GIT_USE_SSH     = true

JOB_INSTALLER_IMAGE   ?=tigera/intrusion-detection-job-installer
BUILD_IMAGES          ?=$(JOB_INSTALLER_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

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
# Building the image
###############################################################################
.PHONY: clean
clean:

# Build the docker image.
.PHONY: $(JOB_INSTALLER_IMAGE) $(JOB_INSTALLER_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(JOB_INSTALLER_IMAGE)
$(JOB_INSTALLER_IMAGE): $(JOB_INSTALLER_IMAGE)-$(ARCH)
$(JOB_INSTALLER_IMAGE)-$(ARCH):
	docker build --pull -t $(JOB_INSTALLER_IMAGE):latest-$(ARCH) --build-arg version=$(BUILD_VERSION) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(JOB_INSTALLER_IMAGE):latest-$(ARCH) $(JOB_INSTALLER_IMAGE):latest
endif

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
ci: image-all

## Deploy images to registry
cd: image-all cd-common

migrate-dashboards:
	bash migrate_dashboards.sh