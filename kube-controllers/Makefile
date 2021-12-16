include ../metadata.mk

PACKAGE_NAME = github.com/projectcalico/calico/kube-controllers
TESLA ?= false

# Makefile configuration options
KUBE_CONTROLLERS_IMAGE  ?=tigera/kube-controllers
FLANNEL_MIGRATION_IMAGE ?=tigera/flannel-migration-controller
DEV_REGISTRIES          ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES      ?=quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

ifdef RELEASE
BUILD_IMAGES ?=$(KUBE_CONTROLLERS_IMAGE)
else
BUILD_IMAGES ?=$(KUBE_CONTROLLERS_IMAGE) $(FLANNEL_MIGRATION_IMAGE)
endif

ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-tesla
	DEV_TAG_SUFFIX        = tesla-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

# Add in local static-checks
LOCAL_CHECKS=check-boring-ssl

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/elasticsearch/ClientBuilder \

ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
# If building on amd64 omit the arch in the container name.
ifeq ($(BUILDARCH),amd64)
	ETCD_IMAGE=quay.io/coreos/etcd:$(ETCD_VERSION)
endif

SRC_FILES=cmd/kube-controllers/main.go $(shell find pkg -name '*.go')

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

###############################################################################

## Removes all build artifacts.
clean:
	rm -rf .go-pkg-cache bin image.created-$(ARCH) build report/*.xml release-notes-*
	-docker rmi $(KUBE_CONTROLLERS_IMAGE)
	-docker rmi $(KUBE_CONTROLLERS_IMAGE):latest-amd64
	-docker rmi $(FLANNEL_MIGRATION_IMAGE)
	-docker rmi $(FLANNEL_MIGRATION_IMAGE):latest-amd64
	rm -f tests/fv/fv.test
	rm -f report/*.xml
	rm -f tests/crds.yaml
	rm -rf tests/crds
	rm -rf vendor

###############################################################################
# Building the binary
###############################################################################
build: bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH)
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/kube-controllers-linux-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) \
	  -e CGO_ENABLED=$(CGO_ENABLED) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS) -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/kube-controllers/'

bin/wrapper-$(ARCH):
	$(DOCKER_RUN) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS)  -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/wrapper'

bin/check-status-linux-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) \
	  -e CGO_ENABLED=$(CGO_ENABLED) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go build $(BUILD_TAGS)  -v -o $@ -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/check-status/'

bin/kubectl-$(ARCH):
	wget https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VERSION)/bin/linux/$(subst armv7,arm,$(ARCH))/kubectl -O $@
	chmod +x $@

###############################################################################
# Building the image
###############################################################################
## Builds the controller binary and docker image.
image: image.created-$(ARCH)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image.created-$(ARCH): bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH) bin/wrapper-$(ARCH) bin/kubectl-$(ARCH)
	# Build the docker image for the policy controller.
	docker build -t $(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f Dockerfile.$(ARCH) .
	# Build the docker image for the flannel migration controller.
	docker build -t $(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f docker-images/flannel-migration/Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	# Need amd64 builds tagged as :latest because Semaphore depends on that
	docker tag $(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) $(KUBE_CONTROLLERS_IMAGE):latest
	docker tag $(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) $(FLANNEL_MIGRATION_IMAGE):latest
endif
	touch $@

.PHONY: remote-deps
remote-deps: mod-download
	@mkdir -p tests/crds/
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c ' \
		cp ../libcalico-go/config/crd/* tests/crds/; \
		chmod +w tests/crds/*'

###############################################################################
# Static checks
###############################################################################
# Make sure that a copyright statement exists on all go files.
check-copyright:
	./check-copyrights.sh

check-boring-ssl: bin/kube-controllers-linux-amd64
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD) \
		go tool nm bin/kube-controllers-linux-amd64 > bin/tags.txt && grep '_Cfunc__goboringcrypto_' bin/tags.txt 1> /dev/null
	-rm -f bin/tags.txt

###############################################################################
# Tests
###############################################################################
## Run the unit tests in a container.
ut:
	$(DOCKER_RUN) --privileged $(CALICO_BUILD) sh -c 'WHAT=$(WHAT) SKIP=$(SKIP) GINKGO_ARGS=$(GINKGO_ARGS) ./run-uts'

.PHONY: fv
## Build and run the FV tests.
fv: remote-deps tests/fv/fv.test image
	@echo Running Go FVs.
	cd tests/fv && ETCD_IMAGE=$(ETCD_IMAGE) \
		KUBE_IMAGE=$(CALICO_BUILD) \
		CONTAINER_NAME=$(KUBE_CONTROLLERS_IMAGE):latest-$(ARCH) \
		MIGRATION_CONTAINER_NAME=$(FLANNEL_MIGRATION_IMAGE):latest-$(ARCH) \
		PRIVATE_KEY=`pwd`/private.key \
		CRDS=$(CURDIR)/tests/crds \
		CERTS=$(CURDIR)/tests/certs \
		GO111MODULE=on \
		./fv.test $(GINKGO_ARGS) -ginkgo.slowSpecThreshold 30

tests/fv/fv.test: $(shell find ./tests -type f -name '*.go' -print)
	# We pre-build the test binary so that we can run it outside a container and allow it
	# to interact with docker.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./tests/fv -c --tags fvtests -o tests/fv/fv.test'

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean mod-download image-all static-checks ut fv

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: cd-common
