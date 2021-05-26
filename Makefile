PACKAGE_NAME    ?= github.com/tigera/calicoq
GO_BUILD_VER    ?= v0.53
GOMOD_VENDOR     = true
GIT_USE_SSH      = true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private
FELIX_REPO       = github.com/tigera/felix-private
TYPHA_REPO       = github.com/tigera/typha-private
LOCAL_CHECKS     = vendor
BINARY           = bin/calicoq

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_CALICOQ_PROJECT_ID)

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Allow local libcalico-go to be mapped into the build container.
ifdef LIBCALICOGO_PATH
EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif

##############################################################################
# Define some constants
##############################################################################
CALICOQ_IMAGE         ?=tigera/calicoq
BUILD_IMAGES          ?=$(CALICOQ_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
ARCHES                ?=amd64
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

BUILD_VER?=latest

CALICOQ_VERSION?=$(shell git describe --tags --dirty --always)
CALICOQ_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CALICOQ_GIT_DESCRIPTION?=$(shell git describe --tags)
CALICOQ_GIT_REVISION?=$(shell git rev-parse --short HEAD)

VERSION_FLAGS=-X $(PACKAGE_NAME)/calicoq/commands.VERSION=$(CALICOQ_VERSION) \
	-X $(PACKAGE_NAME)/calicoq/commands.BUILD_DATE=$(CALICOQ_BUILD_DATE) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_DESCRIPTION=$(CALICOQ_GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_REVISION=$(CALICOQ_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Create an extended go-build image with docker binary installed for use with st-containerized target
TOOLING_IMAGE?=calico/go-build-with-docker
TOOLING_IMAGE_VERSION?=v0.24
TOOLING_IMAGE_CREATED=.go-build-with-docker.created

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

build: $(BINARY)

$(TOOLING_IMAGE_CREATED): Dockerfile-testenv.amd64
	docker build --cpuset-cpus 0 --pull -t $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f Dockerfile-testenv.amd64 .
	touch $@

vendor: go.mod mod-download
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) go mod vendor -v'

.PHONY: ut ut-containerized
ut:
	ginkgo -cover -r --skipPackage vendor calicoq/*

	@echo
	@echo '+==============+'
	@echo '| All coverage |'
	@echo '+==============+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_

	@echo
	@echo '+==================+'
	@echo '| Missing coverage |'
	@echo '+==================+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_ | grep -v '100.0%'

ut-containerized: vendor
	$(DOCKER_RUN) $(CALICO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) make ut'

.PHONY: fv fv-containerized
fv: bin/calicoq
	CALICOQ=`pwd`/$^ fv/run-test

fv-containerized: image run-etcd
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) fv/run-test'

.PHONY: st st-containerized
st: bin/calicoq
	KUBECONFIG=st/kubeconfig CALICOQ=`pwd`/$^ st/run-test

st-containerized: image $(TOOLING_IMAGE_CREATED)
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		-e KUBECONFIG=st/kubeconfig \
		$(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) \
		-c 'CALICOQ=`pwd`/$(BINARY) st/run-test'

.PHONY: scale-test scale-test-containerized
scale-test: bin/calicoq
	CALICOQ=`pwd`/$^ scale-test/run-test

scale-test-containerized: image
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) scale-test/run-test'

# Build image for containerized testing
.PHONY: image $(BUILD_IMAGES)
image: $(BUILD_IMAGES)

# Build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(CALICOQ_IMAGE): binary-containerized
	docker build -t $(CALICOQ_IMAGE):latest-$(ARCH) --file Dockerfile .
ifeq ($(ARCH),amd64)
	docker tag $(CALICOQ_IMAGE):latest-$(ARCH) $(CALICOQ_IMAGE):latest
endif
# Clean up image from containerized testing
.PHONY: clean-image
clean-image:
	docker rmi -f $(shell docker images -a | grep $(CALICOQ_IMAGE) | awk '{print $$3}' | awk '!a[$$0]++')

# All calicoq Go source files.
CALICOQ_GO_FILES:=$(shell find calicoq -type f -name '*.go' -print)

bin/calicoq:
	$(MAKE) binary-containerized

.PHONY: binary-containerized
binary-containerized: $(CALICOQ_GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	mkdir -p .go-pkg-cache bin $(GOMOD_CACHE)
	$(MAKE) vendor
	# Generate the protobuf bindings for Felix
	# Cannot do this together with vendoring since docker permissions in go-build are not perfect?
	$(MAKE) felixbackend
	# Create the binary
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(GIT_CONFIG_SSH) go build -v $(LDFLAGS) -o "$(BINARY)" "./calicoq/calicoq.go"'

###############################################################################
# Updating pins
###############################################################################
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

LICENSING_REPO=github.com/tigera/licensing
LICENSING_BRANCH=$(PIN_BRANCH)
LOGRUS_REPO_ORIG=github.com/sirupsen/logrus
LOGRUS_REPO=github.com/projectcalico/logrus
LOGRUS_BRANCH=$(PIN_BRANCH)

update-licensing-pin:
	$(call update_pin,$(LICENSING_REPO),$(LICENSING_REPO),$(LICENSING_BRANCH))

replace-logrus-pin:
	$(call update_replace_pin,$(LOGRUS_REPO_ORIG),$(LOGRUS_REPO),$(LOGRUS_BRANCH))

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin replace-typha-pin replace-felix-pin update-licensing-pin replace-logrus-pin

###############################################################################
# See .golangci.yml for golangci-lint config
LINT_ARGS +=

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean static-checks fv-containerized ut-containerized st-containerized

## Deploys images to registry
cd: image cd-common

# Generate the protobuf bindings for Felix.
.PHONY: felixbackend
felixbackend: vendor/github.com/projectcalico/felix/proto/felixbackend.proto
	docker run --rm -v `pwd`/vendor/github.com/projectcalico/felix/proto:/src:rw \
	              calico/protoc \
	              --gogofaster_out=. \
	              felixbackend.proto

## Run etcd as a container (calico-etcd)
run-etcd: stop-etcd
	docker run --detach \
	--net=host \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:v3.1.7 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stop the etcd container (calico-etcd)
stop-etcd:
	-docker rm -f calico-etcd

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
