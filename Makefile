PACKAGE_NAME?=github.com/projectcalico/typha
GO_BUILD_VER=v0.53

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_TYPHA_PRIVATE_PROJECT_ID)

# Used so semaphore can trigger the update pin pipelines in projects that have this project as a dependency.
SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_FELIX_PRIVATE_PROJECT_ID) $(SEMAPHORE_CONFD_PRIVATE_PROJECT_ID)

GIT_USE_SSH = true

TYPHA_IMAGE           ?=tigera/typha
BUILD_IMAGES          ?=$(TYPHA_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?= quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

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

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico-go-private in the same directory as this repo.
ifdef LOCAL_BUILD
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go:rw

$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
endif

include Makefile.common

###############################################################################

# Linker flags for building Typha.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
	-X $(PACKAGE_NAME)/pkg/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/pkg/buildinfo.BuildDate=$(DATE) \
	-X $(PACKAGE_NAME)/pkg/buildinfo.GitRevision=$(GIT_COMMIT) \
	-B 0x$(BUILD_ID)"

# All Typha go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_TYPHA_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print)

.PHONY: clean
clean:
	rm -rf .go-pkg-cache \
		bin \
		docker-image/bin \
		build \
		report/*.xml \
		release-notes-* \
		vendor \
		Makefile.common*
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

###############################################################################
# Updating pins
###############################################################################
LIBCALICO_REPO=github.com/tigera/libcalico-go-private

update-pins: update-api-pin update-libcalico-pin

###############################################################################
# Building the binary
###############################################################################
build: bin/calico-typha
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/calico-typha: bin/calico-typha-$(ARCH)
	ln -f bin/calico-typha-$(ARCH) bin/calico-typha

bin/wrapper: bin/wrapper-$(ARCH)
	ln -f bin/wrapper-$(ARCH) bin/wrapper

bin/calico-typha-$(ARCH): $(SRC_FILES) $(LOCAL_BUILD_DEP)
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-typha" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/calico-typha was not statically linked"; false ) )'

bin/wrapper-$(ARCH): $(SRC_FILES) $(LOCAL_BUILD_DEP)
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/wrapper" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/wrapper was not statically linked"; false ) )'

bin/typha-client-$(ARCH): $(SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building typha client...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		GO111MODULE=on go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/typha-client" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/typha-client was not statically linked"; false ) )'

###############################################################################
# Building the image
###############################################################################
# Build the calico/typha docker image, which contains only typha.
.PHONY: $(TYPHA_IMAGE) $(TYPHA_IMAGE)-$(ARCH)
image: $(BUILD_IMAGES)

# Build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

# Build the calico/typha docker image, which contains only Typha.
.PHONY: image $(TYPHA_IMAGE)
$(TYPHA_IMAGE): bin/calico-typha-$(ARCH) bin/wrapper-$(ARCH) register
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/calico-typha-$(ARCH) docker-image/bin/
	cp bin/wrapper-$(ARCH) docker-image/bin/
	cp LICENSE docker-image/
	docker build --pull -t $(TYPHA_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(TYPHA_IMAGE):latest-$(ARCH) $(TYPHA_IMAGE):latest
endif

## tag version number build images i.e.  tigera/typha:latest-amd64 -> tigera/typha:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(TYPHA_IMAGE):latest-$* $(call unescapefs,$(TYPHA_IMAGE):$(VERSION)-$*)


###############################################################################
# Unit Tests
###############################################################################
.PHONY: ut
ut combined.coverprofile: $(SRC_FILES)
	@echo Running Go UTs.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) ./utils/run-coverage'

###############################################################################
# CI/CD
###############################################################################
.PHONY: cd ci version
version: image
	docker run --rm $(TYPHA_IMAGE):latest-$(ARCH) calico-typha --version

ci: mod-download image-all version static-checks ut
ifeq (,$(filter k8sfv-test, $(EXCEPT)))
	@$(MAKE) k8sfv-test
endif

## Avoid unplanned go.sum updates
.PHONY: undo-go-sum check-dirty
undo-go-sum:
	@echo "Undoing go.sum update..."
	git checkout -- go.sum

## Check if generated image is dirty
check-dirty: undo-go-sum
	@if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated image is dirty:"; \
	  git status --porcelain; \
	  false; \
	fi

## Deploys images to registry
cd: check-dirty cd-common

fv: k8sfv-test

k8sfv-test: image
	cd .. && git clone https://github.com/projectcalico/felix.git && cd felix; \
	[ ! -e ../typha/semaphore-felix-branch ] || git checkout $(cat ../typha/semaphore-felix-branch); \
	JUST_A_MINUTE=true USE_TYPHA=true FV_TYPHAIMAGE=$(TYPHA_IMAGE):latest TYPHA_VERSION=latest $(MAKE) k8sfv-test

st:
	@echo "No STs available."

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

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: ut-no-cover
ut-no-cover: $(SRC_FILES)
	@echo Running Go UTs without coverage.
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) ginkgo -r

.PHONY: ut-watch
ut-watch: $(SRC_FILES)
	@echo Watching go UTs for changes...
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) ginkgo watch -r

# Launch a browser with Go coverage stats for the whole project.
.PHONY: cover-browser
cover-browser: combined.coverprofile
	go tool cover -html="combined.coverprofile"

.PHONY: cover-report
cover-report: combined.coverprofile
	# Print the coverage.  We use sed to remove the verbose prefix and trim down
	# the whitespace.
	@echo
	@echo ======== All coverage =========
	@echo
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'go tool cover -func combined.coverprofile | \
				   sed 's=$(PACKAGE_NAME)/==' | \
				   column -t'
	@echo
	@echo ======== Missing coverage only =========
	@echo
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c "go tool cover -func combined.coverprofile | \
				   sed 's=$(PACKAGE_NAME)/==' | \
				   column -t | \
				   grep -v '100\.0%'"

bin/calico-typha.transfer-url: bin/calico-typha-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/calico-typha-$(ARCH) https://transfer.sh/calico-typha > $@'

# Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/onsi/ginkgo/ginkgo
