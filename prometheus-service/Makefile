include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/prometheus-service

##############################################################################
# Constants
##############################################################################
PROMETHEUS_SERVICE_IMAGE  ?=prometheus-service
BUILD_IMAGES          		?=$(PROMETHEUS_SERVICE_IMAGE)

PROMETHEUS_SERVICE_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)

VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

# Create an extended go-build image with docker binary installed for use with st target
TOOLING_IMAGE?=calico/go-build-with-docker
TOOLING_IMAGE_VERSION?=v0.24
TOOLING_IMAGE_CREATED=.go-build-with-docker.created


##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

###############################################################################
# Build
###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "prometheus-service" instead of "bin/prometheus-service".

BUILD_DIR?=build
TOP_SRC_DIRS=pkg
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
SRC_FILES:= $(shell find prometheus-service -type f -name '*.go') \
		$(shell find ../api/pkg -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')


build: bin/prometheus-service-$(ARCH)

bin/prometheus-service-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD_CGO) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" "$(PACKAGE_NAME)/cmd/server"'

# Build the docker image.
.PHONY: $(BUILD_IMAGES) $(addsuffix -$(ARCH),$(BUILD_IMAGES))

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

PROMETHEUS_SERVICE_CONTAINER_CREATED=.$(PROMETHEUS_SERVICE_IMAGE).created-$(ARCH)

image: $(PROMETHEUS_SERVICE_IMAGE)
$(PROMETHEUS_SERVICE_IMAGE): $(PROMETHEUS_SERVICE_CONTAINER_CREATED)
$(PROMETHEUS_SERVICE_CONTAINER_CREATED): bin/prometheus-service-$(ARCH)
	$(DOCKER_BUILD) --pull -t $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

.PHONY: clean
clean:
	rm -rf bin
	-docker image rm -f $$(docker images $(PROMETHEUS_SERVICE_IMAGE) -a -q)

##########################################################################
# Testing
##########################################################################
PROMETHEUS_SERVICE_TEST_CONTAINER_NAME="fv-prometheus-proxy-test"
PACKAGE_ROOT=$(shell sh -c "pwd")

report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go test $(UNIT_TEST_FLAGS) \$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv: image report-dir run-prometheus-service-proxy fv/fv.test
		cd fv && ./fv.test $(GINKGO_ARGS) -ginkgo.slowSpecThreshold 30
		docker rm -f ${PROMETHEUS_SERVICE_TEST_CONTAINER_NAME} || true

fv/fv.test: $(shell find ./fv -type f -name '*.go' -print)
# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
	$(DOCKER_RUN)	$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./fv -c --tags fvtests -o fv/fv.test'
	
run-prometheus-service-proxy:
	docker rm -f ${PROMETHEUS_SERVICE_TEST_CONTAINER_NAME} || true
	${DOCKER_RUN} \
		--detach \
		-v ${PACKAGE_ROOT}/fv/tls.crt:/tls/tls.crt:ro \
		-v ${PACKAGE_ROOT}/fv/tls.key:/tls/tls.key:ro \
		-e LISTEN_ADDR="localhost:8090" \
		-e LOG_LEVEL=debug \
		-e AUTHENTICATION_ENABLED=false \
		--name ${PROMETHEUS_SERVICE_TEST_CONTAINER_NAME} \
		tigera/${PROMETHEUS_SERVICE_IMAGE}:latest
		
	until $$(curl --insecure https://localhost:8090 2> /dev/null); do \
		echo "Waiting for Prometheus Service to come up..."; \
		sleep 3; \
	done

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
.PHONY: ci
## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
ci: clean image-all static-checks ut fv

## Deploys images to registry
cd: image-all cd-common
