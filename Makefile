# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut fv

###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst Dockerfile.%,%,$(wildcard Dockerfile.*))

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
        BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
        BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
        override ARCH=amd64
endif

# Build mounts for running in "local build" mode. Mount in libcalico, but null out
# the vendor directory. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
LOCAL_BUILD_MOUNTS ?=
ifeq ($(LOCAL_BUILD),true)
LOCAL_BUILD_MOUNTS = -v $(CURDIR)/../libcalico-go:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go/vendor:ro
endif

# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

# list of arches *not* to build when doing *-all
#    until s390x works correctly
EXCLUDEARCH ?= s390x
VALIDARCHES = $(filter-out $(EXCLUDEARCH),$(ARCHES))

# Determine which OS.
OS?=$(shell uname -s | tr A-Z a-z)
###############################################################################
GO_BUILD_VER?=v0.20

K8S_VERSION?=v1.14.1
HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION)
ETCD_VERSION?=v3.3.7
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
# If building on amd64 omit the arch in the container name.
ifeq ($(BUILDARCH),amd64)
        ETCD_IMAGE=quay.io/coreos/etcd:$(ETCD_VERSION)
endif

# Makefile configuration options
BUILD_IMAGE?=tigera/kube-controllers
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

PACKAGE_NAME?=github.com/projectcalico/kube-controllers
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
FOSSA_CALICO_BUILD?=calico/go-build:$(FOSSA_GO_BUILD_VER)
LIBCALICOGO_PATH?=none
LOCAL_USER_ID?=$(shell id -u $$USER)

#This is a version with known container with compatible versions of sed/grep etc.
TOOLING_BUILD?=calico/go-build:v0.20

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

# TODO: update all the docker run commands in this make file to use this var
DOCKER_RUN := mkdir -p .go-pkg-cache && \
              docker run --rm \
                         --net=host \
                         $(EXTRA_DOCKER_ARGS) \
                         -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                         -v $${PWD}:/go/src/$(PACKAGE_NAME):rw \
                         -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
                         -w /go/src/$(PACKAGE_NAME)



# Get version from git.
GIT_VERSION?=$(shell git describe --tags --dirty --always)
ifeq ($(LOCAL_BUILD),true)
	GIT_VERSION = $(shell git describe --tags --dirty --always)-dev-build
endif

SRC_FILES=cmd/kube-controllers/main.go $(shell find pkg -name '*.go')

# If local build is set, then always build the binary since we might not
# detect when another local repository has been modified.
ifeq ($(LOCAL_BUILD),true)
.PHONY: $(SRC_FILES)
endif

## Removes all build artifacts.
clean:
	rm -rf bin image.created-$(ARCH)
	-docker rmi $(BUILD_IMAGE)
	-docker rmi $(BUILD_IMAGE):latest-amd64
	rm -f tests/fv/fv.test
	rm -f report/*.xml

###############################################################################
# Building the binary
###############################################################################
build: bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH)
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

# Populates the vendor directory.
vendor: glide.yaml
	# Ensure that the glide cache directory exists.
	mkdir -p $(HOME)/.glide

	# To build without Docker just run "glide install -strip-vendor"
	if [ "$(LIBCALICOGO_PATH)" != "none" ]; then \
          EXTRA_DOCKER_BIND="-v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro"; \
	fi; \
	if [ -n "$(SSH_AUTH_SOCK)" ]; then \
		EXTRA_DOCKER_ARGS="-v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent"; \
	fi; \
	docker run --rm \
		$$EXTRA_DOCKER_ARGS \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw $$EXTRA_DOCKER_BIND \
		-v $(HOME)/.glide:/home/user/.glide:rw \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) glide install -strip-vendor


bin/kube-controllers-linux-$(ARCH): vendor $(SRC_FILES)
	mkdir -p bin
	-mkdir -p .go-pkg-cache
	docker run --rm \
	  -e GOOS=$(OS) -e GOARCH=$(ARCH) \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME):ro \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  -w /go/src/$(PACKAGE_NAME) \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
	  $(LOCAL_BUILD_MOUNTS) \
	  -e GOCACHE=/go-cache \
	  $(CALICO_BUILD) go build -v -o bin/kube-controllers-$(OS)-$(ARCH) -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/kube-controllers/

bin/check-status-linux-$(ARCH): vendor $(SRC_FILES)
	mkdir -p bin
	-mkdir -p .go-pkg-cache
	docker run --rm \
	  -e GOOS=$(OS) -e GOARCH=$(ARCH) \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME):ro \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  -w /go/src/$(PACKAGE_NAME) \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
	  $(LOCAL_BUILD_MOUNTS) \
	  -e GOCACHE=/go-cache \
	  $(CALICO_BUILD) go build -v -o bin/check-status-$(OS)-$(ARCH) -ldflags "-X main.VERSION=$(GIT_VERSION)" ./cmd/check-status/

###############################################################################
# Building the image
###############################################################################
## Builds the controller binary and docker image.
image: image.created-$(ARCH)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image.created-$(ARCH): bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH)
	# Build the docker image for the policy controller.
	docker build -t $(BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	# Need amd64 builds tagged as :latest because Semaphore depends on that
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif
	touch $@

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))
sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
	# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*:$(IMAGETAG))-ARCH --target $(call unescapefs,$*:$(IMAGETAG))"

## push default amd64 arch where multi-arch manifest is not supported
push-non-manifests: imagetag $(addprefix sub-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))
sub-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker push $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of one arch
tag-images: imagetag $(addprefix sub-single-tag-images-arch-,$(call escapefs,$(PUSH_IMAGES))) $(addprefix sub-single-tag-images-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))
sub-single-tag-images-arch-%:
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(VALIDARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

## tag version number build images i.e.  tigera/kube-controllers:latest-amd64 -> tigera/kube-controllers:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)


###############################################################################
# Managing the upstream library pins
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-felix-pin update-licensing-pin

## deprecated target alias
update-libcalico: update-pins
	$(warning !! Update update-libcalico is deprecated, use update-pins !!)


## deprecated target alias
update-felix: update-pins
	$(warning !! Update update-felix is deprecated, use update-pins !!)


## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;


###############################################################################
## felix

## Set the default FELIX source for this project
FELIX_PROJECT_DEFAULT=tigera/felix-private.git
FELIX_GLIDE_LABEL=projectcalico/felix

## Default the FELIX repo and version but allow them to be overridden (master or release-vX.Y)
## default FELIX branch to the same branch name as the current checked out repo
FELIX_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
FELIX_REPO?=github.com/$(FELIX_PROJECT_DEFAULT)
FELIX_VERSION?=$(shell git ls-remote git@github.com:$(FELIX_PROJECT_DEFAULT) $(FELIX_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure FELIX repo and branch are reachable
guard-git-felix:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "$(FELIX_BRANCH)" "Ensure the branch exists, or set FELIX_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(FELIX_PROJECT_DEFAULT) "$(FELIX_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(FELIX_VERSION))" = "" ]; then \
		echo "ERROR: FELIX version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-felix-pin: guard-ssh-forwarding-bug guard-git-felix
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(FELIX_GLIDE_LABEL)" \
		REPO="$(FELIX_REPO)" \
		VERSION="$(FELIX_VERSION)" \
		DEFAULT_REPO="$(FELIX_PROJECT_DEFAULT)" \
		BRANCH="$(FELIX_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '

###############################################################################
## licensing

## Set the default LICENSING source for this project
LICENSING_PROJECT_DEFAULT=tigera/licensing
LICENSING_GLIDE_LABEL=tigera/licensing

## Default the LICENSING repo and version but allow them to be overridden (master or release-vX.Y)
## default LICENSING branch to the same branch name as the current checked out repo
LICENSING_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LICENSING_REPO?=github.com/$(LICENSING_PROJECT_DEFAULT)
LICENSING_VERSION?=$(shell git ls-remote git@github.com:$(LICENSING_PROJECT_DEFAULT) $(LICENSING_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure LICENSING repo and branch are reachable
guard-git-licensing:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "$(LICENSING_BRANCH)" "Ensure the branch exists, or set LICENSING_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "$(LICENSING_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(LICENSING_VERSION))" = "" ]; then \
		echo "ERROR: LICENSING version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-licensing-pin: guard-ssh-forwarding-bug guard-git-licensing
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(LICENSING_GLIDE_LABEL)" \
		REPO="$(LICENSING_REPO)" \
		VERSION="$(LICENSING_VERSION)" \
		DEFAULT_REPO="$(LICENSING_PROJECT_DEFAULT)" \
		BRANCH="$(LICENSING_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '




###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
## Perform static checks on the code.
static-checks: vendor check-copyright
	docker run --rm \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) gometalinter --deadline=300s --disable-all --enable=goimports --enable=vet --enable=errcheck --vendor -s test_utils ./...

.PHONY: fix
## Fix static checks
fix goimports:
	goimports -l -w ./pkg
	goimports -l -w ./cmd/kube-controllers/main.go
	goimports -l -w ./cmd/check-status/main.go
	goimports -l -w ./tests

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

# Make sure that a copyright statement exists on all go files.
check-copyright:
	./check-copyrights.sh

foss-checks: vendor
	@echo Running $@...
	@docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(CALICO_BUILD) /usr/local/bin/fossa

###############################################################################
# Tests
###############################################################################
## Run the unit tests in a container.
ut: vendor
	-mkdir -p .go-pkg-cache
	docker run --rm --privileged --net=host \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $(CURDIR)/.go-pkg-cache:/go/pkg/:rw \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) sh -c 'WHAT=$(WHAT) SKIP=$(SKIP) GINKGO_ARGS="$(GINKGO_ARGS)" ./run-uts'

.PHONY: fv
## Build and run the FV tests.
fv: tests/fv/fv.test image
	@echo Running Go FVs.
	cd tests/fv && ETCD_IMAGE=$(ETCD_IMAGE) \
		HYPERKUBE_IMAGE=$(HYPERKUBE_IMAGE) \
		CONTAINER_NAME=$(BUILD_IMAGE):latest-$(ARCH) \
		PRIVATE_KEY=`pwd`/private.key \
		CRDS_FILE=${PWD}/vendor/github.com/projectcalico/libcalico-go/test/crds.yaml \
		./fv.test $(GINKGO_ARGS) -ginkgo.slowSpecThreshold 30

tests/fv/fv.test: $(shell find ./tests -type f -name '*.go' -print)
	# We pre-build the test binary so that we can run it outside a container and allow it
	# to interact with docker.
	mkdir -p .go-pkg-cache && \
		docker run --rm \
		--net=host \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-v $${PWD}:/go/src/$(PACKAGE_NAME):rw \
		-v $${PWD}/.go-pkg-cache:/go/pkg:rw \
		$(LOCAL_BUILD_MOUNTS) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) go test ./tests/fv -c --tags fvtests -o tests/fv/fv.test

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean image-all static-checks ut fv

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=${BRANCH_NAME} EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) tag-base-images-all
	$(MAKE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish"
	@echo ""

## Produces a git tag for the release.
release-tag: release-prereqs release-notes
	git tag $(VERSION) -F release-notes-$(VERSION)
	@echo ""
	@echo "Now you can build the release:"
	@echo ""
	@echo "  make VERSION=$(VERSION) release-build"
	@echo ""

## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif

	$(MAKE) image-all
	$(MAKE) tag-images-all IMAGETAG=$(VERSION)
	# Generate the `latest` images.
	$(MAKE) tag-images-all IMAGETAG=latest

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	if ! docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) --version | grep '^$(VERSION)$$'; then echo "Reported version:" `docker run $(BUILD_IMAGE):$(VERSION)-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	# Push images.
	$(MAKE) push-all push-manifests push-non-manifests IMAGETAG=$(VERSION)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://github.com/tigera/kube-controllers/releases/tag/$(VERSION)"
	@echo ""
	@echo "If this is the latest stable release, then run the following to push 'latest' images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish-latest"
	@echo ""

# WARNING: Only run this target if this release is the latest stable release. Do NOT
# run this target for alpha / beta / release candidate builds, or patches to earlier Calico versions.
## Pushes `latest` release images. WARNING: Only run this for latest stable releases.
release-publish-latest: release-prereqs
	# Check latest versions match.
	if ! docker run $(BUILD_IMAGE):latest --version | grep '^$(VERSION)$$'; then echo "Reported version:" `docker run $(BUILD_IMAGE):latest --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

	$(MAKE) push-all push-manifests push-non-manifests IMAGETAG=latest

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: help
## Display this help text.
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	$(info Available targets)
	@awk '/^[a-zA-Z\-\_0-9\/]+:/ {                                      \
		nb = sub( /^## /, "", helpMsg );                                \
		if(nb == 0) {                                                   \
			helpMsg = $$0;                                              \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                   \
		}                                                               \
		if (nb)                                                         \
			printf "\033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;  \
	}                                                                   \
	{ helpMsg = $$0 }'                                                  \
	width=20                                                            \
	$(MAKEFILE_LIST)
