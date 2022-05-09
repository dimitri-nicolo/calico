include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/anomaly-detection-api

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Build
###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "anomaly-detection-api" instead of "$(BINDIR)/anomaly-detection-api".
ANOMALY_DETECTION_API_IMAGE ?=anomaly-detection-api
BUILD_IMAGES        ?=$(ANOMALY_DETECTION_API_IMAGE)


build: anomaly-detection-api

BINDIR?=bin
BUILD_DIR?=build
TOP_SRC_DIRS=pkg
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
SRC_FILES= $(shell sh -c "find pkg cmd -name \\*.go")

anomaly-detection-api: $(BINDIR)/anomaly-detection-api

$(BINDIR)/anomaly-detection-api: $(BINDIR)/anomaly-detection-api-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd $(BINDIR) && ln -s -T anomaly-detection-api-$(ARCH) anomaly-detection-api'

$(BINDIR)/anomaly-detection-api-$(ARCH): $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd" && \
				( ldd $(BINDIR)/anomaly-detection-api-$(ARCH) 2>&1 | \
	                grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $(BINDIR)/anomaly-detection-api-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(BUILD_IMAGES) $(addsuffix -$(ARCH),$(BUILD_IMAGES))

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(BUILD_IMAGES)
image: $(BUILD_IMAGES)
$(ANOMALY_DETECTION_API_IMAGE): $(ANOMALY_DETECTION_API_IMAGE)-$(ARCH)
$(ANOMALY_DETECTION_API_IMAGE)-$(ARCH): $(BINDIR)/anomaly-detection-api-$(ARCH)
	docker build --pull -t $(ANOMALY_DETECTION_API_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ANOMALY_DETECTION_API_IMAGE):latest-$(ARCH) $(ANOMALY_DETECTION_API_IMAGE):latest
endif

##########################################################################
# Testing
##########################################################################
AD_API_FV_TEST_CONTAINER_NAME ?="fv-ad-api-test"
PACKAGE_ROOT=$(shell sh -c "pwd")

report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url.ssh://git@github.com.insteadOf https://github.com && \
			go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv: report-dir run-ad-image-fv fv/fv.test
	cd fv && ./fv.test $(GINKGO_ARGS) -ginkgo.slowSpecThreshold 30
	docker rm -f ${AD_API_FV_TEST_CONTAINER_NAME} || true

fv/fv.test: $(shell find ./fv -type f -name '*.go' -print)
# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./fv -c --tags fvtests -o fv/fv.test'

run-ad-image-fv:
	docker rm -f ${AD_API_FV_TEST_CONTAINER_NAME} || true
	$(DOCKER_RUN) \
	  --detach \
		-v ${PACKAGE_ROOT}/fv/resources/tls/tls.crt:/tls/tls.crt:ro \
		-v ${PACKAGE_ROOT}/fv/resources/tls/tls.key:/tls/tls.key:ro \
		-e LISTEN_ADDR="localhost:8080" \
		-e DEBUG_RBAC_DISABLED="true" \
		-e STORAGE_PATH="./fv/temp" \
		-e LOG_LEVEL=debug \
		--name ${AD_API_FV_TEST_CONTAINER_NAME} \
		${ANOMALY_DETECTION_API_IMAGE}

###############################################################################
# Static checks
###############################################################################
# See .golangci.yml for golangci-lint config
# SA1019 are deprecation checks, we don't want to fail on those because it means a library update that deprecates something
# requires immediate removing of the deprecated functions.
LINT_ARGS += --exclude SA1019

###############################################################################
# CI/CD
###############################################################################

ci: clean image static-checks ut fv

## Deploys images to registry
cd: image-all cd-common

.PHONY: clean
clean:
	docker rmi -f $(ANOMALY_DETECTION_API_IMAGE):latest > /dev/null 2>&1
	docker rmi -f $(ANOMALY_DETECTION_API_IMAGE):latest-$(ARCH) > /dev/null 2>&1
	docker rm -f ${AD_API_FV_TEST_CONTAINER_NAME}  > /dev/null 2>&1
	rm -rf $(BINDIR) .go-pkg-cache Makefile.common*Test targets