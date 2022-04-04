include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/calicoq

##############################################################################
# Define some constants
##############################################################################
CALICOQ_IMAGE         ?=calicoq
BUILD_IMAGES          ?=$(CALICOQ_IMAGE)

RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

VERSION_FLAGS=-X $(PACKAGE_NAME)/calicoq/commands.VERSION=$(GIT_VERSION) \
	-X $(PACKAGE_NAME)/calicoq/commands.BUILD_DATE=$(DATE) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_REVISION=$(GIT_VERSION)

BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

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

# All calicoq Go source files.
SRC_FILES:=$(shell find calicoq -type f -name '*.go' -print)

# Build image for testing
.PHONY: image $(BUILD_IMAGES)
image: $(BUILD_IMAGES)

# Build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

BINARY = bin/calicoq
$(CALICOQ_IMAGE): $(BINARY)
	$(DOCKER_BUILD) -t $(CALICOQ_IMAGE):latest-$(ARCH) --file Dockerfile . --load
ifeq ($(ARCH),amd64)
	docker tag $(CALICOQ_IMAGE):latest-$(ARCH) $(CALICOQ_IMAGE):latest
endif
# Clean up image from testing
.PHONY: clean-image
clean-image:
	docker rmi -f $(shell docker images -a | grep $(CALICOQ_IMAGE) | awk '{print $$3}' | awk '!a[$$0]++')

build: $(BINARY)
$(BINARY): $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	mkdir -p .go-pkg-cache bin $(GOMOD_CACHE)

	# Create the binary
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(GIT_CONFIG_SSH) go build -v $(LDFLAGS) -o "$(BINARY)" "./calicoq/calicoq.go"'

###############################################################################
# CI/CD
###############################################################################
# Location of a valid license to use in the tests. On semaphore, this is provided as part
# of the build environment.
TSEE_TEST_LICENSE?=${HOME}/secrets/new-test-customer-license.yaml

.PHONY: ci
## Run what CI runs
ci: clean static-checks fv ut st

## Deploys images to registry
cd: image cd-common

$(TOOLING_IMAGE_CREATED): Dockerfile-testenv.amd64
	$(DOCKER_BUILD) --cpuset-cpus 0 -t $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f Dockerfile-testenv.amd64 . --load
	touch $@

.PHONY: ut 
ut:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) ginkgo -cover -r --skipPackage calicoq/*'

.PHONY: fv 
fv: image run-etcd
	$(DOCKER_RUN) \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(TSEE_TEST_LICENSE):$(TSEE_TEST_LICENSE) \
		-e ETCD_IMAGE=$(ETCD_IMAGE) \
		-e TSEE_TEST_LICENSE=$(TSEE_TEST_LICENSE) \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) fv/run-test'

.PHONY: st
st: image $(TOOLING_IMAGE_CREATED)
	$(DOCKER_RUN) \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(TSEE_TEST_LICENSE):$(TSEE_TEST_LICENSE) \
		-w /code/$(PACKAGE_NAME) \
		-e KUBECONFIG=st/kubeconfig \
		-e ETCD_IMAGE=$(ETCD_IMAGE) \
		-e TSEE_TEST_LICENSE=$(TSEE_TEST_LICENSE) \
		$(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) \
		-c 'CALICOQ=`pwd`/$(BINARY) st/run-test'

.PHONY: scale-test
scale-test: image
	$(DOCKER_RUN) \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		-e ETCD_IMAGE=$(ETCD_IMAGE) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) scale-test/run-test'

.PHONY: clean
clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release vendor
	-docker rmi calico/build
	-docker rmi $(CALICOQ_IMAGE) -f
	-docker rmi $(CALICO_BUILD) -f
	-docker rmi $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f
	-rm -f $(TOOLING_IMAGE_CREATED)

###############################################################################
# Release
###############################################################################
release-verify-version: var-require-all-VERSION
ifdef CONFIRM
	$(eval CURRENT_RELEASE_VERSION := $(git-release-tag-for-current-commit))
	$(if $(CURRENT_RELEASE_VERSION),,echo Current commit has not been tagged with a release version && exit 1)
	$(if $(filter $(VERSION),$(git-release-tag-for-current-commit)),,\
		echo Current version $(CURRENT_RELEASE_VERSION) does not match given version $(VERSION) && exit 1)
endif

## Builds and pushed binaries to the public s3 bucket.
release-publish-binaries: var-require-one-of-CONFIRM-DRYRUN var-require-all-VERSION release-verify-version build
ifdef CONFIRM
	aws --profile helm s3 cp bin/calicoq s3://tigera-public/ee/binaries/$(VERSION)/calicoq --acl public-read
else
	@echo [DRYRUN] aws --profile helm s3 bin/cp calicoq s3://tigera-public/ee/binaries/$(VERSION)/calicoq --acl public-read
endif
