###############################################################################
# Subcomponents
###############################################################################
.PHONY: config-snapshotter-release-build
config-snapshotter-release-build:
	$(MAKE) -f Makefile.config-snapshotter release-build

.PHONY: report-generator-release-build
report-generator-release-build:
	$(MAKE) -f Makefile.report-generator release-build

.PHONY: report-generator-scheduler-release-build
report-generator-scheduler-release-build:
	$(MAKE) -f Makefile.report-generator-scheduler release-build

.PHONY: config-snapshotter-release-verify
config-snapshotter-release-verify:
	$(MAKE) -f Makefile.config-snapshotter release-verify

.PHONY: report-generator-release-verify
report-generator-release-verify:
	$(MAKE) -f Makefile.report-generator release-verify

.PHONY: report-generator-scheduler-release-verify
report-generator-scheduler-release-verify:
	$(MAKE) -f Makefile.report-generator-scheduler release-verify

.PHONY: config-snapshotter-push-all
config-snapshotter-push-all:
	$(MAKE) -f Makefile.config-snapshotter push-all

.PHONY: report-generator-push-all
report-generator-push-all:
	$(MAKE) -f Makefile.report-generator push-all

.PHONY: report-generator-scheduler-push-all
report-generator-scheduler-push-all:
	$(MAKE) -f Makefile.report-generator-scheduler push-all

.PHONY: tigera/config-snapshotter
tigera/config-snapshotter:
	$(MAKE) -f Makefile.config-snapshotter tigera/config-snapshotter

.PHONY: tigera/report-generator
tigera/report-generator:
	$(MAKE) -f Makefile.report-generator tigera/report-generator

.PHONY: tigera/report-generator-scheduler
tigera/report-generator-scheduler:
	$(MAKE) -f Makefile.report-generator-scheduler tigera/report-generator-scheduler

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty)
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
release-build: release-prereqs config-snapshotter-release-build report-generator-release-build report-generator-scheduler-release-build
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs config-snapshotter-release-verify report-generator-release-verify report-generator-scheduler-release-verify

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

push-all: config-snapshotter-push-all report-generator-push-all report-generator-scheduler-push-all
