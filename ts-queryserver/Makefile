##############################################################################
# Copyright 2019-2022 Tigera Inc. All rights reserved.
##############################################################################
include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/ts-queryserver

#############################################
# Env vars related to packaging and releasing
#############################################
QUERYSERVER_IMAGE ?=cnx-queryserver
BUILD_IMAGES      ?=$(QUERYSERVER_IMAGE)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
# All queryserver Go source files.
SRC_FILES = $(shell find queryserver pkg -type f -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../calicoctl -name '*.go') \
		$(shell find ../felix -name '*.go') \
		$(shell find ../lma/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../licensing/client -name '*.go')

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/cnx-queryserver-$(ARCH)

.PHONY: bin/cnx-queryserver-$(ARCH)
bin/cnx-queryserver-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" queryserver/$*/*.go'

###############################################################################
# BUILD IMAGE
###############################################################################
QUERYSERVER_CONTAINER_CREATED=.cnx-queryserver.created-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(QUERYSERVER_IMAGE)

$(QUERYSERVER_IMAGE): $(QUERYSERVER_CONTAINER_CREATED)
$(QUERYSERVER_CONTAINER_CREATED): Dockerfile.$(ARCH) bin/cnx-queryserver-$(ARCH)
	$(DOCKER_BUILD) -t $(QUERYSERVER_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -rf bin
	rm -f $(QUERYSERVER_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(QUERYSERVER_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
#############################################
# Run unit level tests
#############################################
.PHONY: ut
## Run only Unit Tests.
ut:
	echo "running UTs"
	./run-uts

.PHONY: ut-containerized
## Run UT in a Docker container
ut-containerized: run-etcd
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'make WHAT=$(WHAT) SKIP=$(SKIP) ut'

#############################################
# Run FV tests
#############################################
.PHONY: fv
fv:
	echo "running FVs"
	./run-fvs

# test targets used by build
.PHONY: fv-containerized
## Run FV in a Docker container
fv-containerized: run-etcd
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'make WHAT=$(WHAT) SKIP=$(SKIP) fv'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean image static-checks ut-containerized fv-containerized

## Deploys images to registry
cd: image cd-common
