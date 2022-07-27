PACKAGE_NAME?=github.com/tigera/eck-operator-docker
GO_BUILD_VER?=v0.73

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_ECK_OPERATOR_DOCKER_PROJECT_ID)

ECK_OPERATOR_NAME     ?= eck-operator
ECK_OPERATOR_IMAGE    ?=tigera/$(ECK_OPERATOR_NAME)
BUILD_IMAGES          ?=$(ECK_OPERATOR_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

GO_VERSION  ?=1.17.9
UBI_VERSION ?=8.5

VERSION ?= $(shell cat cloud-on-k8s/VERSION)
LDFLAGS ?= "-X github.com/elastic/cloud-on-k8s/pkg/about.version=$(VERSION) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildHash=$(shell cd cloud-on-k8s && git rev-parse --short=8 --verify HEAD) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildSnapshot=false"

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
else
CGO_ENABLED=0
endif

build: bin/$(ECK_OPERATOR_NAME)-$(ARCH)

bin/$(ECK_OPERATOR_NAME)-$(ARCH): prepare-build
	$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			cd cloud-on-k8s && go build -o ../$@ -v -ldflags $(LDFLAGS) cmd/main.go'

image: $(ECK_OPERATOR_IMAGE)
$(ECK_OPERATOR_IMAGE): $(ECK_OPERATOR_IMAGE)-$(ARCH)
$(ECK_OPERATOR_IMAGE)-$(ARCH): build
	docker build $(DOCKER_SQUASH) -t $(ECK_OPERATOR_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ECK_OPERATOR_IMAGE):latest-$(ARCH) $(ECK_OPERATOR_IMAGE):latest
endif

prepare-build:
	git submodule update --init --recursive
	$(DOCKER_GO_BUILD) bash -x prepare-build.sh

.PHONY: cd
cd: image cd-common

.PHONY: clean
clean:
	rm -rf bin \
		   .go-pkg-cache \
		   Makefile.*
