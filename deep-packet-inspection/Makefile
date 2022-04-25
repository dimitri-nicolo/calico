# Copyright 2021-2022 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/deep-packet-inspection

#############################################
# Env vars related to packaging and releasing
#############################################
DEEP_PACKET_INSPECTION_IMAGE	?=deep-packet-inspection
SNORT_IMAGE      		?=snort3
SNORT3_VERSION  		?=3.1.21.0
SNORT_DEV_REGISTRY 		?=gcr.io/unique-caldron-775/cnx/tigera
BUILD_IMAGES			?=$(DEEP_PACKET_INSPECTION_IMAGE)

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES = $(shell find pkg cmd -name '*.go') $(shell find ../libcalico-go/lib -name '*.go') $(shell find ../lma/pkg -name '*.go')

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/deep-packet-inspection-$(ARCH)

.PHONY: bin/deep-packet-inspection-$(ARCH)
bin/deep-packet-inspection-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go && \
				( ldd $@ 2>&1 | \
					grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# BUILD IMAGE
###############################################################################
# Build the docker image.
.PHONY: $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(SNORT_IMAGE)-$(ARCH) $(DEEP_PACKET_INSPECTION_IMAGE)-$(ARCH)
image: $(BUILD_IMAGES)

$(SNORT_IMAGE)-$(ARCH):
	if docker manifest inspect $(SNORT_DEV_REGISTRY)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH); then \
  		echo "Using existing snort image $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH)"; \
  		docker pull $(SNORT_DEV_REGISTRY)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH); \
  		docker tag $(SNORT_DEV_REGISTRY)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH); \
  	else \
  	  	echo "Snort image  $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) doesn't exist in $(SNORT_DEV_REGISTRY), building it"; \
  	  	$(DOCKER_BUILD) --build-arg SNORT3_VERSION=$(SNORT3_VERSION) -t $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) -f docker-image/Dockerfile.snort.$(ARCH) docker-image --load; \
  	fi

$(DEEP_PACKET_INSPECTION_IMAGE): $(SNORT_IMAGE)-$(ARCH) $(DEEP_PACKET_INSPECTION_IMAGE)-$(ARCH)
$(DEEP_PACKET_INSPECTION_IMAGE)-$(ARCH): bin/deep-packet-inspection-$(ARCH)
	# can't use $(DOCKER_BUILD) from lib.Makefile due to the --pull flag.
	# it will load metadata for docker.io/tigera/snort3 which causes an internal error.
	docker buildx build --build-arg SNORT3_VERSION=$(SNORT3_VERSION) -t $(DEEP_PACKET_INSPECTION_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

.PHONY: push-snort-image
push-snort-image:
	docker push $(SNORT_DEV_REGISTRY)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH)

.PHONY: clean
clean:
	rm -rf bin \
		report/*.xml \
		config/
	find . -name '*.coverprofile' -type f -delete
	-docker image rm -f $$(docker images $(DEEP_PACKET_INSPECTION_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(SNORT_IMAGE) -a -q)

###############################################################################
# Testing
###############################################################################
MOCKERY_FILE_PATHS= \
	pkg/processor/Processor \
	pkg/exec/Exec \
	pkg/dispatcher/Dispatcher \
	pkg/eventgenerator/EventGenerator \
	pkg/elastic/Client \
	pkg/elastic/ESForwarder \
	pkg/dpiupdater/DPIStatusUpdater \
	pkg/file/FileMaintainer

GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################
GINKGO_FOCUS?=Syncer

# Comma separated paths to packages containing fv tests
FV_PACKAGE=pkg/syncer

.PHONY: ut
## Run only Unit Tests.
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download && $(GINKGO) -r -skipPackage=$(FV_PACKAGE) pkg/*'


###############################################################################
# FV Tests
###############################################################################
## Run the ginkgo FVs
fv: run-k8s-apiserver
	 $(DOCKER_RUN) -e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		ginkgo -r --focus="$(GINKGO_FOCUS)" $(GINKGO_ARGS)'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut fv

## Deploys images to registry
cd: image-all push-snort-image cd-common
