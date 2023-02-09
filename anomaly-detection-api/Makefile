include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/anomaly-detection-api

ANOMALY_DETECTION_API_IMAGE ?=anomaly-detection-api
BUILD_IMAGES ?= $(ANOMALY_DETECTION_API_IMAGE)

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

BINDIR?=bin
TEST_DIRS ?= $(shell find pkg -name '*_test.go' -exec dirname {} \; | sort | uniq)
SRC_FILES = $(shell find pkg cmd -name '*.go') \
	$(shell find ../crypto/pkg -name '*.go') \
	$(shell find ../lma/pkg -name '*.go')
AD_API_FV_TEST_CONTAINER_NAME ?=fv-ad-api-test

###############################################################################
# Build the binary
###############################################################################
.PHONY: build
build: $(BINDIR)/anomaly-detection-api-$(ARCH)

.PHONY: bin/anomaly-detection-api-$(ARCH)
$(BINDIR)/anomaly-detection-api-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) cmd/main.go'

###############################################################################
# Build the image
###############################################################################
ANOMALY_DETECTION_API_CONTAINER_CREATED=.anomaly-detection-api.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(ANOMALY_DETECTION_API_IMAGE)

$(ANOMALY_DETECTION_API_IMAGE): $(ANOMALY_DETECTION_API_CONTAINER_CREATED)
$(ANOMALY_DETECTION_API_CONTAINER_CREATED): Dockerfile.$(ARCH) $(BINDIR)/anomaly-detection-api-$(ARCH)
	$(DOCKER_BUILD) -t $(ANOMALY_DETECTION_API_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -rf $(BINDIR)
	rm -f $(ANOMALY_DETECTION_API_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(ANOMALY_DETECTION_API_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(AD_API_FV_TEST_CONTAINER_NAME) -a -q)

##########################################################################
# Testing
##########################################################################
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

gen-crt-key:
	openssl req -x509 -nodes -newkey rsa:2048 -keyout ${PACKAGE_ROOT}/fv/resources/tls/tls.key -out ${PACKAGE_ROOT}/fv/resources/tls/tls.crt \
		-subj "/CN=localhost" \
		-addext "extendedKeyUsage = serverAuth" \
		-addext "keyUsage = digitalSignature" \
		-addext "subjectAltName = DNS:localhost"

run-ad-image-fv: image gen-crt-key
	docker rm -f ${AD_API_FV_TEST_CONTAINER_NAME} || true
	$(DOCKER_RUN) \
		--detach \
		-v ${PACKAGE_ROOT}/fv/resources/tls/tls.crt:/tls/tls.crt:ro \
		-v ${PACKAGE_ROOT}/fv/resources/tls/tls.key:/tls/tls.key:ro \
		-e LISTEN_ADDR="localhost:8080" \
		-e DEBUG_RBAC_DISABLED="true" \
		-e STORAGE_PATH="./fv/temp" \
		-e LOG_LEVEL=debug \
		-u ${LOCAL_USER_ID} \
		--name ${AD_API_FV_TEST_CONTAINER_NAME} \
		${ANOMALY_DETECTION_API_IMAGE}:latest-$(ARCH)

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

ci: clean static-checks ut fv

## Deploys images to registry
cd: image-all cd-common
