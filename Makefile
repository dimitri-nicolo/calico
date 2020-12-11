###############################################################################
# Subcomponents
###############################################################################
.PHONY: install-release-build
install-release-build:
	$(MAKE) -C install release-build

.PHONY: controller-release-build
controller-release-build:
	$(MAKE) -C controller release-build

.PHONY: install-release-verify
install-release-verify:
	$(MAKE) -C install release-verify

.PHONY: controller-release-verify
controller-release-verify:
	$(MAKE) -C controller release-verify

.PHONY: install-push-all
install-push-all:
	$(MAKE) -C install push-all

.PHONY: controller-push-all
controller-push-all:
	$(MAKE) -C controller push-all

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
ifndef VERSION
	BUILD_VERSION = $(GIT_VERSION)
else
	BUILD_VERSION = $(VERSION)
endif

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make IMAGETAG=$(VERSION) RELEASE=true push-all"
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
release-build: release-prereqs install-release-build controller-release-build
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs install-release-verify controller-release-verify

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif

push-all: install-push-all controller-push-all
