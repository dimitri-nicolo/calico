PACKAGE_NAME?=github.com/tigera/eck-operator-docker
GO_BUILD_VER?=v0.85

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ECK_OPERATOR_DOCKER_PROJECT_ID)

ECK_OPERATOR_NAME     ?= eck-operator
ECK_OPERATOR_IMAGE    ?=tigera/$(ECK_OPERATOR_NAME)
BUILD_IMAGES          ?=$(ECK_OPERATOR_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

ECK_OPERATOR_VERSION = 2.6.1

VERSION ?= $(shell cat cloud-on-k8s/VERSION)
LDFLAGS ?= -X github.com/elastic/cloud-on-k8s/pkg/about.version=$(VERSION) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildHash=$(shell cd cloud-on-k8s && git rev-parse --short=8 --verify HEAD) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildSnapshot=false

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
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
# Build
###############################################################################
ECK_OPERATOR_DOWNLOADED=.eck-operator.downloaded

# Add --squash argument for CICD pipeline runs only to avoid setting "experimental",
# for Docker processes on personal machine.
# DOCKER_SQUASH is defaulted to be empty but can be set `DOCKER_SQUASH=--squash make image` 
# to squash images locally.
ifdef CI
DOCKER_SQUASH=--squash
endif

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
GOEXPERIMENT=boringcrypto
TAGS=osusergo,netgo
else
CGO_ENABLED=0
endif

.PHONY: init-source
init-source: $(ECK_OPERATOR_DOWNLOADED)
$(ECK_OPERATOR_DOWNLOADED):
	mkdir -p cloud-on-k8s
	curl -sfL https://github.com/elastic/cloud-on-k8s/archive/refs/tags/$(ECK_OPERATOR_VERSION).tar.gz | tar xz --strip-components 1 -C cloud-on-k8s
	patch -d cloud-on-k8s -p1 < patches/0001-Upgrade-golang.org-x-packages.patch
	touch $@

build: bin/$(ECK_OPERATOR_NAME)-$(ARCH)

bin/$(ECK_OPERATOR_NAME)-$(ARCH): $(ECK_OPERATOR_DOWNLOADED)
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) -e GOEXPERIMENT=$(GOEXPERIMENT) $(CALICO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			cd cloud-on-k8s && \
			make go-generate && \
			make generate-config-file && \
			go build -buildvcs=false -o ../$@ -v -tags $(TAGS) -ldflags "$(LDFLAGS) -linkmode external -extldflags -static -s -w" cmd/main.go'
ifeq ($(ARCH), $(filter $(ARCH),amd64))
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'strings bin/eck-operator-$(ARCH) | grep '_Cfunc__goboringcrypto_' 1> /dev/null'
endif

.PHONY: clean
clean:
	rm -fr bin/ cloud-on-k8s/
	rm -f $(ECK_OPERATOR_DOWNLOADED)
	-docker image rm -f $$(docker images $(ECK_OPERATOR_IMAGE) -a -q)

###############################################################################
# Image
###############################################################################
.PHONY: image
image: $(ECK_OPERATOR_IMAGE)
$(ECK_OPERATOR_IMAGE): $(ECK_OPERATOR_IMAGE)-$(ARCH)
$(ECK_OPERATOR_IMAGE)-$(ARCH): build
	docker buildx build --pull $(DOCKER_SQUASH) -t $(ECK_OPERATOR_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ECK_OPERATOR_IMAGE):latest-$(ARCH) $(ECK_OPERATOR_IMAGE):latest
endif

.PHONY: cd
cd: image cd-common
