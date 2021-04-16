PACKAGE_NAME=github.com/projectcalico/kube-controllers
GO_BUILD_VER=v0.51

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID=$(SEMAPHORE_KUBE_CONTROLLERS_PRIVATE_PROJECT_ID)

GIT_USE_SSH = true

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
# assuming that there is a local checkout of libcalico in the same directory as this repo.
ifdef LOCAL_BUILD
.PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go:rw \
	-v $(CURDIR)/../felix-private:/go/src/github.com/projectcalico/felix:rw \
	-v $(CURDIR)/../typha-private:/go/src/github.com/projectcalico/typha:rw

$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go \
		-replace=github.com/projectcalico/felix=../felix \
		-replace=github.com/projectcalico/typha=../typha
endif

# Makefile configuration options
BUILD_IMAGE?=calico/kube-controllers
FLANNEL_MIGRATION_BUILD_IMAGE?=calico/flannel-migration-controller
PUSH_IMAGES?=$(BUILD_IMAGE) quay.io/$(BUILD_IMAGE) $(FLANNEL_MIGRATION_BUILD_IMAGE) quay.io/$(FLANNEL_MIGRATION_BUILD_IMAGE)
RELEASE_IMAGES?=

include Makefile.common

###############################################################################

KUBE_CONTROLLERS_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/elasticsearch/ClientBuilder \

HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION)
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
# If building on amd64 omit the arch in the container name.
ifeq ($(BUILDARCH),amd64)
	ETCD_IMAGE=quay.io/coreos/etcd:$(ETCD_VERSION)
endif

SRC_FILES=cmd/kube-controllers/main.go $(shell find pkg -name '*.go')

## Removes all build artifacts.
clean:
	rm -rf .go-pkg-cache bin image.created-$(ARCH) build report/*.xml release-notes-*
	-docker rmi $(BUILD_IMAGE)
	-docker rmi $(BUILD_IMAGE):latest-amd64
	-docker rmi $(FLANNEL_MIGRATION_BUILD_IMAGE)
	-docker rmi $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-amd64
	rm -f tests/fv/fv.test
	rm -f report/*.xml
	rm -f tests/crds.yaml
	rm -rf tests/crds
	rm -rf vendor
	rm Makefile.common*

###############################################################################
# Updating pins
###############################################################################
LICENSING_BRANCH?=$(PIN_BRANCH)
LICENSING_REPO?=github.com/tigera/licensing
TIGERA_API_BRANCH?=$(PIN_BRANCH)
TIGERA_API_REPO?=github.com/tigera/api
LIBCALICO_REPO=github.com/tigera/libcalico-go-private
FELIX_REPO=github.com/tigera/felix-private
TYPHA_REPO=github.com/tigera/typha-private
CNI_REPO=github.com/tigera/cni-plugin-private

update-licensing-pin:
	$(call update_pin,github.com/tigera/licensing,$(LICENSING_REPO),$(LICENSING_BRANCH))

update-tigerapi-pin:
	$(call update_pin,github.com/tigera/api,$(TIGERA_API_REPO),$(TIGERA_API_BRANCH))

update-pins: update-licensing-pin replace-libcalico-pin replace-felix-pin replace-typha-pin replace-cni-pin update-tigerapi-pin	

###############################################################################
# Building the binary
###############################################################################
build: bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH)
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/kube-controllers-linux-$(ARCH): $(LOCAL_BUILD_DEP) $(SRC_FILES)
	$(DOCKER_RUN) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	  go build -v -o $@ -ldflags "-X main.VERSION=$(KUBE_CONTROLLERS_VERSION)" ./cmd/kube-controllers/'

bin/wrapper-$(ARCH):
	$(DOCKER_RUN) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	  go build -v -o $@ -ldflags "-X main.VERSION=$(KUBE_CONTROLLERS_VERSION)" ./cmd/wrapper'

bin/check-status-linux-$(ARCH): $(LOCAL_BUILD_DEP) $(SRC_FILES)
	$(DOCKER_RUN) \
	  -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin \
	  $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	  go build -v -o $@ -ldflags "-X main.VERSION=$(KUBE_CONTROLLERS_VERSION)" ./cmd/check-status/'

bin/kubectl-$(ARCH):
	wget https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VERSION)/bin/linux/$(ARCH)/kubectl -O $@
	chmod +x $@

###############################################################################
# Building the image
###############################################################################
## Builds the controller binary and docker image.
image: image.created-$(ARCH)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image.created-$(ARCH): bin/kube-controllers-linux-$(ARCH) bin/check-status-linux-$(ARCH) bin/wrapper-$(ARCH) bin/kubectl-$(ARCH)
	# Build the docker image for the policy controller.
	docker build -t $(BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f Dockerfile.$(ARCH) .
	# Build the docker image for the flannel migration controller.
	docker build -t $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) -f docker-images/flannel-migration/Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	# Need amd64 builds tagged as :latest because Semaphore depends on that
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
	docker tag $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) $(FLANNEL_MIGRATION_BUILD_IMAGE):latest
endif
	touch $@

.PHONY: remote-deps
remote-deps: mod-download
	@mkdir -p tests/crds/
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c ' \
		cp `go list -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/config/crd/* tests/crds/; \
		chmod +w tests/crds/*'

sub-single-tag-images-arch-%:
	@if echo $* | grep -q "$(call escapefs,$(FLANNEL_MIGRATION_BUILD_IMAGE))"; then \
		echo "docker tag $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))"; \
		docker tag $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH)); \
	else \
		echo "docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))"; \
		docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH)); \
	fi

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	@if echo $* | grep -q "$(call escapefs,$(FLANNEL_MIGRATION_BUILD_IMAGE))"; then \
		echo "docker tag $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))"; \
		docker tag $(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)); \
	else \
		echo "docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))"; \
		docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)); \
	fi
else
	$(NOECHO) $(NOOP)
endif

## tag version number build images i.e.  tigera/kube-controllers:latest-amd64 -> tigera/kube-controllers:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)

###############################################################################
# Static checks
###############################################################################
# Make sure that a copyright statement exists on all go files.
check-copyright:
	./check-copyrights.sh

###############################################################################
# Tests
###############################################################################
## Run the unit tests in a container.
ut: $(LOCAL_BUILD_DEP)
	$(DOCKER_RUN) --privileged $(CALICO_BUILD) sh -c 'WHAT=$(WHAT) SKIP=$(SKIP) GINKGO_ARGS=$(GINKGO_ARGS) ./run-uts'

.PHONY: fv
## Build and run the FV tests.
fv: remote-deps tests/fv/fv.test image
	@echo Running Go FVs.
	cd tests/fv && ETCD_IMAGE=$(ETCD_IMAGE) \
		HYPERKUBE_IMAGE=$(HYPERKUBE_IMAGE) \
		CONTAINER_NAME=$(BUILD_IMAGE):latest-$(ARCH) \
		MIGRATION_CONTAINER_NAME=$(FLANNEL_MIGRATION_BUILD_IMAGE):latest-$(ARCH) \
		PRIVATE_KEY=`pwd`/private.key \
		CRDS=${PWD}/tests/crds \
		GO111MODULE=on \
		./fv.test $(GINKGO_ARGS) -ginkgo.slowSpecThreshold 30

tests/fv/fv.test: $(LOCAL_BUILD_DEP) $(shell find ./tests -type f -name '*.go' -print)
	# We pre-build the test binary so that we can run it outside a container and allow it
	# to interact with docker.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./tests/fv -c --tags fvtests -o tests/fv/fv.test'

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean mod-download image-all static-checks ut fv

## Avoid unplanned go.sum updates
.PHONY: undo-go-sum check-dirty
undo-go-sum:
	@if (git status --porcelain go.sum | grep -o 'go.sum'); then \
	  @echo "Undoing go.sum update..."; \
	  git checkout -- go.sum; \
	fi

## Check if generated image is dirty
check-dirty: undo-go-sum
	@if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated image is dirty:"; \
	  git status --porcelain; \
	  false; \
	fi

###############################################################################
# CD
###############################################################################
.PHONY: cd
## Deploys images to registry
cd: check-dirty cd-common

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
