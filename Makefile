.PHONY: all test

default: all
all: test
test: ut

BUILD_VER?=latest
BUILD_IMAGE:=tigera/calicoq
REGISTRY_PREFIX?=gcr.io/unique-caldron-775/cnx/
PACKAGE_NAME?=github.com/tigera/calicoq
LOCAL_USER_ID?=$(shell id -u $$USER)
BINARY:=bin/calicoq

GO_BUILD_VER?=v0.30
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
# Specific version for fossa license checks
FOSSA_GO_BUILD_VER?=v0.18
FOSSA_GO_BUILD?=calico/go-build:$(FOSSA_GO_BUILD_VER)

CALICOQ_VERSION?=$(shell git describe --tags --dirty --always)
CALICOQ_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CALICOQ_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CALICOQ_GIT_DESCRIPTION?=$(shell git describe --tags)

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

##### XXX: temp changes from common Makefile ####
BUILD_OS ?= $(shell uname -s | tr A-Z a-z)
BUILD_ARCH ?= $(shell uname -m)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
    BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
    BUILDARCH=amd64
endif

ARCH ?= $(BUILDARCH)
# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
    override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
    override ARCH=amd64
endif

GOMOD_VENDOR := true
ifeq ($(GOMOD_VENDOR),true)
    GOFLAGS?="-mod=vendor"
else
ifeq ($(CI),true)
ifneq ($(LOCAL_BUILD),true)
    GOFLAGS?="-mod=readonly"
endif
endif
endif

ifneq ($(GOPATH),)
    GOMOD_CACHE = $(shell echo $(GOPATH) | cut -d':' -f1)/pkg/mod
else
    # If gopath is empty, default to $(HOME)/go.
    GOMOD_CACHE = $(HOME)/go/pkg/mod
endif

EXTRA_DOCKER_ARGS += -e GO111MODULE=on -v $(GOMOD_CACHE):/go/pkg/mod:rw

GIT_CONFIG_SSH ?= git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"

define get_remote_version
    $(shell git ls-remote https://$(1) $(2) 2>/dev/null | cut -f 1)
endef

# update_pin updates the given package's version to the latest available in the specified repo and branch.
# $(1) should be the name of the package, $(2) and $(3) the repository and branch from which to update it.
define update_pin
    $(eval new_ver := $(call get_remote_version,$(2),$(3)))

    $(DOCKER_RUN) $(CALICO_BUILD) sh -c '\
        if [[ ! -z "$(new_ver)" ]]; then \
            $(GIT_CONFIG_SSH); \
            go get $(1)@$(new_ver); \
            go mod download; \
        else \
            error "error getting remote version"; \
        fi'
endef

# update_replace_pin updates the given package's version to the latest available in the specified repo and branch.
# This routine can only be used for packages being replaced in go.mod, such as private versions of open-source packages.
# $(1) should be the name of the package, $(2) and $(3) the repository and branch from which to update it.
define update_replace_pin
    $(eval new_ver := $(call get_remote_version,$(2),$(3)))

    $(DOCKER_RUN) -i $(CALICO_BUILD) sh -c '\
        if [ ! -z "$(new_ver)" ]; then \
            $(GIT_CONFIG_SSH); \
            go mod edit -replace $(1)=$(2)@$(new_ver); \
            go mod download; \
        else \
            echo "error getting remote version"; \
            exit 1; \
        fi'
endef

PIN_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LIBCALICO_BRANCH?=$(PIN_BRANCH)
LIBCALICO_REPO?=github.com/tigera/libcalico-go-private
FELIX_BRANCH?=$(PIN_BRANCH)
FELIX_REPO?=github.com/tigera/felix-private
LOGRUS_BRANCH?=$(PIN_BRANCH)
LOGRUS_REPO?=github.com/projectcalico/logrus
LICENSING_BRANCH?=$(PIN_BRANCH)
LICENSING_REPO?=github.com/tigera/licensing

update-libcalico-pin:
	$(call update_replace_pin,github.com/projectcalico/libcalico-go,$(LIBCALICO_REPO),$(LIBCALICO_BRANCH))

update-felix-pin:
	$(call update_replace_pin,github.com/projectcalico/felix,$(FELIX_REPO),$(FELIX_BRANCH))

update-logrus-pin:
	$(call update_replace_pin,github.com/sirupsen/logrus,$(LOGRUS_REPO),$(LOGRUS_BRANCH))

update-licensing-pin:
	$(call update_pin,$(LICENSING_REPO),$(LICENSING_REPO),$(LICENSING_BRANCH))
##### temp changes from common Makefile #####

$(TOOLING_IMAGE_CREATED): Dockerfile-testenv.amd64
	docker build --cpuset-cpus 0 --pull -t $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f Dockerfile-testenv.amd64 .
	touch $@

# Volume-mount gopath into the build container to cache go module's packages. If the environment is using multiple
# comma-separated directories for gopath, use the first one, as that is the default one used by go modules.
ifneq ($(GOPATH),)
    # If the environment is using multiple comma-separated directories for gopath, use the first one, as that
    # is the default one used by go modules.
    GOMOD_CACHE = $(shell echo $(GOPATH) | cut -d':' -f1)/pkg/mod
else
    # If gopath is empty, default to $(HOME)/go.
    GOMOD_CACHE = $$HOME/go/pkg/mod
endif

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

DOCKER_RUN := mkdir -p .go-pkg-cache bin $(GOMOD_CACHE) && \
                docker run --rm \
                         --net=host \
                         $(EXTRA_DOCKER_ARGS) \
                         -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                         -e GOCACHE=/go-cache \
                         -e GOPATH=/go \
                         -e GOOS=$(BUILD_OS) \
                         -e GOARCH=$(ARCH) \
                         -e GOFLAGS=$(GOFLAGS) \
                         -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
                         -v $(CURDIR)/.go-pkg-cache:/go-cache:rw \
                         -w /go/src/$(PACKAGE_NAME)

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

vendor: go.mod go.sum
	$(DOCKER_RUN) $(CALICO_BUILD) go mod vendor -v

foss-checks: vendor
	@echo Running $@...
	@docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(FOSSA_GO_BUILD) /usr/local/bin/fossa

.PHONY: ut
ut: bin/calicoq
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

.PHONY: ut-containerized
ut-containerized: bin/calicoq
	docker run --rm -t \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		$(CALICO_BUILD) \
		sh -c 'make ut'

.PHONY: fv
fv: bin/calicoq
	CALICOQ=`pwd`/$^ fv/run-test

.PHONY: fv-containerized
fv-containerized: build-image run-etcd
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) fv/run-test'

.PHONY: st
st: bin/calicoq
	CALICOQ=`pwd`/$^ st/run-test

.PHONY: st-containerized
st-containerized: build-image $(TOOLING_IMAGE_CREATED)
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) \
		-c 'CALICOQ=`pwd`/$(BINARY) st/run-test'

.PHONY: scale-test
scale-test: bin/calicoq
	CALICOQ=`pwd`/$^ scale-test/run-test

.PHONY: scale-test-containerized
scale-test-containerized: build-image
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) scale-test/run-test'

# Build image for containerized testing
.PHONY: build-image
build-image: bin/calicoq
	docker build -t $(BUILD_IMAGE):$(BUILD_VER) `pwd`

# Clean up image from containerized testing
.PHONY: clean-image
clean-image:
	docker rmi -f $(shell docker images -a | grep $(BUILD_IMAGE) | awk '{print $$3}' | awk '!a[$$0]++')

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
	   sh -c '$(GIT_CONFIG_SSH) && \
	   GOPRIVATE="github.com/tigera" cd /go/src/github.com/tigera/calicoq && \
	   go build -v $(LDFLAGS) -o "$(BINARY)" "./calicoq/calicoq.go"'

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

tag-image: imagetag build-image
	docker tag $(BUILD_IMAGE):latest $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(IMAGETAG)

push-image: imagetag tag-image
	docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(IMAGETAG)



###############################################################################
# Managing the upstream library pins
#
# If you're updating the pins with a non-release branch checked out,
# set PIN_BRANCH to the parent branch, e.g.:
#
#     PIN_BRANCH=release-v2.5 make update-pins
#        - or -
#     PIN_BRANCH=master make update-pins
#
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-libcalico-pin update-felix-pin update-logrus-pin update-licensing-pin

## TODO: Not sure if this is still needed.
## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

###############################################################################
# Static checks
###############################################################################
# TODO: re-enable these linters !
LINT_ARGS := --disable gosimple,govet,structcheck,errcheck,goimports,unused,ineffassign,staticcheck,deadcode,typecheck

.PHONY: static-checks
## Perform static checks on the code.
static-checks: vendor
	$(DOCKER_RUN) \
	  $(CALICO_BUILD) \
	  golangci-lint run --deadline 5m $(LINT_ARGS)

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean static-checks fv-containerized ut-containerized st-containerized

## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) push-image IMAGETAG=${BRANCH_NAME}


.PHONY: release
release: clean clean-release release/calicoq

release/calicoq: $(CALICOQ_GO_FILES) clean
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=v.X.Y.Z)
endif
	git tag $(VERSION)

	# Check to make sure the tag isn't "dirty"
	if git describe --tags --dirty | grep dirty; \
	then echo current git working tree is "dirty". Make sure you do not have any uncommitted changes ;false; fi

	# Build the calicoq binaries and image
	$(MAKE) binary-containerized RELEASE_BUILD=1
	$(MAKE) build-image

	# Make the release directory and move over the relevant files
	mkdir -p release
	mv $(BINARY) release/calicoq-$(CALICOQ_GIT_DESCRIPTION)
	ln -f release/calicoq-$(CALICOQ_GIT_DESCRIPTION) release/calicoq

	# Check that the version output includes the version specified.
	# Tests that the "git tag" makes it into the binaries. Main point is to catch "-dirty" builds
	# Release is currently supported on darwin / linux only.
	if ! docker run $(BUILD_IMAGE) version | grep 'Version:\s*$(VERSION)$$'; then \
	  echo "Reported version:" `docker run $(BUILD_IMAGE) version` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

	# Retag images with correct version and registry prefix
	docker tag $(BUILD_IMAGE) $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)

	# Check that images were created recently and that the IDs of the versioned and latest images match
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(BUILD_IMAGE)
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)

	@echo "\nNow push the tag and images."
	@echo "git push origin $(VERSION)"
	@echo "gcloud auth configure-docker"
	@echo "docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)"
	@echo "\nIf this release version is the newest stable release, also tag and push the"
	@echo "images with the 'latest' tag"
	@echo "docker tag $(BUILD_IMAGE) $(REGISTRY_PREFIX)$(BUILD_IMAGE):latest"
	@echo "docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):latest"

.PHONY: compress-release
compressed-release: release/calicoq
	# Requires "upx" to be in your PATH.
	# Compress the executable with upx.  We get 4:1 compression with '-8'; the
	# more agressive --best gives a 0.5% improvement but takes several minutes.
	upx -8 release/calicoq-$(CALICOQ_GIT_DESCRIPTION)
	ln -f release/calicoq-$(CALICOQ_GIT_DESCRIPTION) release/calicoq

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

.PHONY: clean-release
clean-release:
	-rm -rf release

.PHONY: clean
clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release vendor
	-docker rmi calico/build
	-docker rmi $(BUILD_IMAGE) -f
	-docker rmi $(CALICO_BUILD) -f
	-docker rmi $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f
	rm -f $(TOOLING_IMAGE_CREATED)
