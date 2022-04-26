###############################################################################
# Subcomponents
###############################################################################
.PHONY: snapshotter-release-build
snapshotter-release-build:
	$(MAKE) -f Makefile.snapshotter release-build

.PHONY: reporter-release-build
reporter-release-build:
	$(MAKE) -f Makefile.reporter release-build

.PHONY: controller-release-build
controller-release-build:
	$(MAKE) -f Makefile.controller release-build

.PHONY: server-release-build
server-release-build:
	$(MAKE) -f Makefile.server release-build

.PHONY: snapshotter-release-verify
snapshotter-release-verify:
	$(MAKE) -f Makefile.snapshotter release-verify

.PHONY: reporter-release-verify
reporter-release-verify:
	$(MAKE) -f Makefile.reporter release-verify

.PHONY: controller-release-verify
controller-release-verify:
	$(MAKE) -f Makefile.controller release-verify

.PHONY: server-release-verify
server-release-verify:
	$(MAKE) -f Makefile.server release-verify

.PHONY: snapshotter-push-all
snapshotter-push-all:
	$(MAKE) -f Makefile.snapshotter push-all

.PHONY: reporter-push-all
reporter-push-all:
	$(MAKE) -f Makefile.reporter push-all

.PHONY: controller-push-all
controller-push-all:
	$(MAKE) -f Makefile.controller push-all

.PHONY: server-push-all
server-push-all:
	$(MAKE) -f Makefile.server push-all

.PHONY: tigera/compliance-snapshotter
tigera/compliance-snapshotter:
	$(MAKE) -f Makefile.snapshotter tigera/compliance-snapshotter

.PHONY: tigera/compliance-reporter
tigera/compliance-reporter:
	$(MAKE) -f Makefile.reporter tigera/compliance-reporter

.PHONY: tigera/compliance-controller
tigera/compliance-controller:
	$(MAKE) -f Makefile.controller tigera/compliance-controller

.PHONY: tigera/compliance-server
tigera/compliance-server:
	$(MAKE) -f Makefile.server tigera/compliance-server

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty --abbrev=12)
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
release-build: release-prereqs snapshotter-release-build reporter-release-build controller-release-build server-release-build
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs snapshotter-release-verify reporter-release-verify controller-release-verify server-release-verify

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

push-all: snapshotter-push-all reporter-push-all controller-push-all server-push-all

.PHONY: ci
ci:
	$(MAKE) -f Makefile.server ci
	$(MAKE) -f Makefile.reporter ci
	$(MAKE) -f Makefile.snapshotter ci
	$(MAKE) -f Makefile.controller ci

.PHONY: cd
cd:
	$(MAKE) -f Makefile.server cd
	$(MAKE) -f Makefile.reporter cd
	$(MAKE) -f Makefile.snapshotter cd
	$(MAKE) -f Makefile.controller cd

# Run fossa.io license checks
foss-checks:
	@echo Running $@...
	@docker run --rm -v $(CURDIR)/../:/code/:rw \
	  -e LOCAL_USER_ID=$(MY_UID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /code/ \
	  $(CALICO_BUILD) /usr/local/bin/fossa
