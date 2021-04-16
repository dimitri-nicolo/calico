PACKAGE_NAME?=github.com/tigera/egress-gateway
GO_BUILD_VER?=v0.51
GIT_USE_SSH=true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_EGRESS_GATEWAY_PROJECT_ID)

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

EXTRA_DOCKER_ARGS+=-e GOPRIVATE='github.com/tigera/*'

include Makefile.common

###############################################################################
CNX_REPOSITORY?=gcr.io/unique-caldron-775/cnx
BUILD_IMAGE?=tigera/egress-gateway
PUSH_IMAGES?=$(CNX_REPOSITORY)/tigera/egress-gateway
RELEASE_IMAGES?=

# Variables controlling the image
GATEWAY_CONTAINER_CREATED=.egress_gateway.created-$(ARCH)

# Files that go into the image
GATEWAY_CONTAINER_FILES=$(shell find ./filesystem -type f)

## Clean enough that a new release build will be clean
clean:
	find . -name '*.created' -exec rm -f {} +
	find . -name '*.pyc' -exec rm -f {} +
	rm -rf .go-pkg-cache
	rm -rf certs *.tar
	rm -rf dist
	rm -rf Makefile.common*
	# Delete images that we built in this repo
	docker rmi $(BUILD_IMAGE):latest-$(ARCH) || true

###############################################################################
# Building the image
###############################################################################
## Create the image for the current ARCH
image: $(BUILD_IMAGE)
## Create the images for all supported ARCHes
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(BUILD_IMAGE): $(GATEWAY_CONTAINER_CREATED)
$(GATEWAY_CONTAINER_CREATED): register ./Dockerfile.$(ARCH) $(GATEWAY_CONTAINER_FILES)
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) . --build-arg GIT_VERSION=$(GIT_VERSION) -f ./Dockerfile.$(ARCH)
	touch $@

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean image-all

## Deploys images to registry
cd: cd-common

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) CALICO_VERSION=$(CALICO_VERSION_RELEASE) VERSION=$(VERSION) release-tag
	$(MAKE) CALICO_VERSION=$(CALICO_VERSION_RELEASE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) tag-base-images-all
	$(MAKE) CALICO_VERSION=$(CALICO_VERSION_RELEASE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make CALICO_VERSION=$(CALICO_VERSION_RELEASE) VERSION=$(VERSION) release-publish"
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
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=$(VERSION)
	# Generate the `latest` images.
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=latest

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	echo "" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	# Push images.
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=$(VERSION)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "If this is the latest stable release, then run the following to push 'latest' images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish-latest"
	@echo ""

# WARNING: Only run this target if this release is the latest stable release. Do NOT
# run this target for alpha / beta / release candidate builds, or patches to earlier Calico versions.
## Pushes `latest` release images. WARNING: Only run this for latest stable releases.
release-publish-latest: release-verify
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=latest

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifndef CALICO_VERSION_RELEASE
	$(error CALICO_VERSION_RELEASE is undefined - run using make release CALICO_VERSION_RELEASE=vX.Y.Z)
endif

## tag version number build images i.e.  tigera/egress-gateway:latest-amd64 -> tigera/egress-gateway:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)

$(info "Calico git version")
$(info $(shell printf "%-21s = %-10s\n" "GIT_VERSION" $(GIT_VERSION)))
