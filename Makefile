# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut

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
BUILDOS ?= $(shell uname -s | tr A-Z a-z)

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

# Determine which OS.
OS := $(shell uname -s | tr A-Z a-z)

###############################################################################
GO_BUILD_VER?=v0.16
ETCD_VER?=v3.3.7

CARROTCTL_VERSION?=$(shell git describe --tags --dirty --always)
CARROTCTL_DIR=carrotctl
CONTAINER_NAME?=calico/carrotctl
CARROTCTL_FILES=$(shell find $(CARROTCTL_DIR) -name '*.go')
CTL_CONTAINER_CREATED=$(CARROTCTL_DIR)/.calico_ctl.created-$(ARCH)

TEST_CONTAINER_NAME ?= calico/test

CARROTCTL_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CARROTCTL_GIT_REVISION?=$(shell git rev-parse --short HEAD)
GIT_VERSION?=$(shell git describe --tags --dirty)

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

GO_BUILD_CONTAINER?=calico/go-build:$(GO_BUILD_VER)-$(BUILDARCH)
LOCAL_USER_ID?=$(shell id -u $$USER)

PACKAGE_NAME?=github.com/tigera/licensing

LDFLAGS=-ldflags "-X $(PACKAGE_NAME)/carrotctl/cmd.VERSION=$(CARROTCTL_VERSION) \
	-X $(PACKAGE_NAME)/carrotctl/cmd.BUILD_DATE=$(CARROTCTL_BUILD_DATE) \
	-X $(PACKAGE_NAME)/carrotctl/cmd.GIT_REVISION=$(CARROTCTL_GIT_REVISION) -s -w"

LIBCALICOGO_PATH?=none

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	rm -rf bin build certs *.tar vendor
	docker rmi $(CONTAINER_NAME):latest-$(ARCH) || true
	docker rmi $(CONTAINER_NAME):$(VERSION)-$(ARCH) || true
ifeq ($(ARCH),amd64)
	docker rmi $(CONTAINER_NAME):latest || true
	docker rmi $(CONTAINER_NAME):$(VERSION) || true
endif

###############################################################################
# Building the binary
###############################################################################
.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/carrotctl-linux-,$(ARCHES)) bin/carrotctl-windows-amd64.exe bin/carrotctl-darwin-amd64

.PHONY: build
## Build the binary for the current architecture and platform
build: bin/carrotctl-$(OS)-$(ARCH)

## Create the vendor directory
vendor: Gopkg.toml
	mkdir -p .go-pkg-cache
	docker run --rm -i \
	  $(EXTRA_DOCKER_ARGS) \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(GO_BUILD_CONTAINER) dep ensure


# The supported different binary names. For each, ensure that an OS and ARCH is set
bin/carrotctl-%-amd64: ARCH=amd64
bin/carrotctl-%-arm64: ARCH=arm64
bin/carrotctl-%-ppc64le: ARCH=ppc64le
bin/carrotctl-%-s390x: ARCH=s390x
bin/carrotctl-darwin-amd64: OS=darwin
bin/carrotctl-windows-amd64: OS=windows
bin/carrotctl-linux-%: OS=linux

bin/carrotctl-%: $(CARROTCTL_FILES) vendor
	mkdir -p bin
	-mkdir -p .go-pkg-cache
	docker run --rm -i \
	  -e OS=$(OS) -e ARCH=$(ARCH) \
	  -e GOOS=$(OS) -e GOARCH=$(ARCH) \
	  -e CARROTCTL_VERSION=$(CARROTCTL_VERSION) \
	  -e CARROTCTL_BUILD_DATE=$(CARROTCTL_BUILD_DATE) -e CARROTCTL_GIT_REVISION=$(CARROTCTL_GIT_REVISION) \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME):ro \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -v $(CURDIR)/.go-pkg-cache:/go-cache/:rw \
	  -e GOCACHE=/go-cache \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(GO_BUILD_CONTAINER) go build -v -o bin/carrotctl-$(OS)-$(ARCH) $(LDFLAGS) "./carrotctl/carrotctl.go"

# Overrides for the binaries that need different output names
bin/carrotctl: bin/carrotctl-linux-amd64
	cp $< $@
bin/carrotctl-windows-amd64.exe: bin/carrotctl-windows-amd64
	mv $< $@

###############################################################################
# Building the image
###############################################################################
.PHONY: image calico/carrotctl
image: calico/carrotctl
calico/carrotctl: $(CTL_CONTAINER_CREATED)
$(CTL_CONTAINER_CREATED): Dockerfile.$(ARCH) bin/carrotctl-linux-$(ARCH)
	docker build -t $(CONTAINER_NAME):latest-$(ARCH) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(CONTAINER_NAME):latest-$(ARCH) $(CONTAINER_NAME):latest
endif
	touch $@

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag
	docker push $(CONTAINER_NAME):$(IMAGETAG)-$(ARCH)
	docker push quay.io/$(CONTAINER_NAME):$(IMAGETAG)-$(ARCH)
ifeq ($(ARCH),amd64)
	docker push $(CONTAINER_NAME):$(IMAGETAG)
	docker push quay.io/$(CONTAINER_NAME):$(IMAGETAG)
endif

push-all: imagetag $(addprefix sub-push-,$(ARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## tag images of one arch
tag-images: imagetag
	docker tag $(CONTAINER_NAME):latest-$(ARCH) $(CONTAINER_NAME):$(IMAGETAG)-$(ARCH)
	docker tag $(CONTAINER_NAME):latest-$(ARCH) quay.io/$(CONTAINER_NAME):$(IMAGETAG)-$(ARCH)
ifeq ($(ARCH),amd64)
	docker tag $(CONTAINER_NAME):latest-$(ARCH) $(CONTAINER_NAME):$(IMAGETAG)
	docker tag $(CONTAINER_NAME):latest-$(ARCH) quay.io/$(CONTAINER_NAME):$(IMAGETAG)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(ARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

###############################################################################
# Static checks
###############################################################################
## Perform static checks on the code.
.PHONY: static-checks
static-checks: vendor
	docker run --rm \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(GO_BUILD_CONTAINER) gometalinter --deadline=300s --disable-all --enable=goimports --vendor ./...

.PHONY: fix
## Fix static checks
fix:
	goimports -w carrotctl/*

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev.
ut: bin/carrotctl-linux-amd64
	docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
    -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
    $(GO_BUILD_CONTAINER) sh -c 'cd /go/src/$(PACKAGE_NAME) && go test ./...'


###############################################################################
# CI
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean build static-checks ut calico/carrotctl

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
	$(MAKE) tag-images push IMAGETAG=${BRANCH_NAME}
	$(MAKE) tag-images push IMAGETAG=$(shell git describe --tags --dirty --always --long)

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
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

	$(MAKE) image
	$(MAKE) tag-images IMAGETAG=$(VERSION)
	# Generate the `latest` images.
	$(MAKE) tag-images IMAGETAG=latest

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	if ! docker run $(CONTAINER_NAME):$(VERSION)-$(ARCH) version | grep 'Version:\s*$(VERSION)$$'; then \
	  echo "Reported version:" `docker run $(CONTAINER_NAME):$(VERSION)-$(ARCH) version` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

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
	$(MAKE) push IMAGETAG=$(VERSION) ARCH=$(ARCH)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "Attach the following binaries to the release."
	@echo ""
	@echo "- bin/carrotctl"
	@echo "- bin/carrotctl-linux-amd64"
	@echo "- bin/carrotctl-linux-arm64"
	@echo "- bin/carrotctl-linux-ppc64le"
	@echo "- bin/carrotctl-linux-s390x"
	@echo "- bin/carrotctl-darwin-amd64"
	@echo "- bin/carrotctl-windows-amd64.exe"
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
	if ! docker run $(CONTAINER_NAME):latest-$(ARCH) version | grep 'Version:\s*$(VERSION)$$'; then \
	  echo "Reported version:" `docker run $(CONTAINER_NAME):latest-$(ARCH) version` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

	$(MAKE) push IMAGETAG=latest ARCH=$(ARCH)

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: help
## Display this help text
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	@echo "carrotctl Makefile"
	@echo
	@echo "Dependencies: docker 1.12+; go 1.8+"
	@echo
	@echo "For some target, set ARCH=<target> OS=<os> to build for a given target architecture and OS."
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
	@echo
	@echo "-----------------------------------------"
	@echo "Building for $(OS)-$(ARCH) INSTALL_FLAG=$(INSTALL_FLAG)"
	@echo
	@echo "ARCH (target):          $(ARCH)"
	@echo "OS (target):            $(OS)"
	@echo "BUILDARCH (host):       $(BUILDARCH)"
	@echo "GO_BUILD_CONTAINER:     $(GO_BUILD_CONTAINER)"
	@echo "-----------------------------------------"
