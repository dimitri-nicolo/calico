PACKAGE_NAME=github.com/projectcalico/cni-plugin
GO_BUILD_VER=v0.51

GIT_USE_SSH = true

# This excludes deprecation checks. libcalico-go-private has deprecated VethNameForWorkload but libcalico-go has not, so
# to the drift between public and private to a minimum we just disable the deprecation checks.
LINT_ARGS = --max-issues-per-linter 0 --max-same-issues 0 --deadline 5m --exclude SA1019

# This needs to be evaluated before the common makefile is included.
# This var contains some default values that the common makefile may append to.
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/tigera/cni

BUILD_IMAGE_ORG?=calico
CALICO_BUILD?=$(BUILD_IMAGE_ORG)/go-build:$(GO_BUILD_VER)

BUILD_IMAGE?=tigera/cni
DEPLOY_CONTAINER_MARKER=cni_deploy_container-$(ARCH).created
RELEASE_IMAGES?=

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_CNI_PLUGIN_PRIVATE_PROJECT_ID)

# Used so semaphore can trigger the update pin pipelines in projects that have this project as a dependency.
SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_NODE_PRIVATE_PROJECT_ID)

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
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go:/go/src/github.com/projectcalico/libcalico-go:rw
$(LOCAL_BUILD_DEP):
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
endif

include Makefile.common

###############################################################################
BIN=bin/$(ARCH)
SRC_FILES=$(shell find pkg cmd internal -name '*.go')
TEST_SRC_FILES=$(shell find tests -name '*.go')
WINFV_SRCFILES=$(shell find win_tests -name '*.go')
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')

# fail if unable to download
CURL=curl -C - -sSf

CNI_VERSION=v0.8.6

# By default set the CNI_SPEC_VERSION to 0.3.1 for tests.
CNI_SPEC_VERSION?=0.3.1

DEPLOY_CONTAINER_MARKER=cni_deploy_container-$(ARCH).created

ETCD_CONTAINER ?= quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
# If building on amd64 omit the arch in the container name.
ifeq ($(BUILDARCH),amd64)
	ETCD_CONTAINER=quay.io/coreos/etcd:$(ETCD_VERSION)
endif

.PHONY: clean
clean:
	rm -rf $(BIN) bin $(DEPLOY_CONTAINER_MARKER) .go-pkg-cache pkg/install/install.test 
	rm -f *.created
	rm -f crds.yaml
	rm -rf config/
	# If the vendor directory exists then the build fails with golang 1.14, and this folder exists in semaphore v1 but it's
	# empty
	rm -rf vendor/
	rm Makefile.common*

###############################################################################
# Updating pins
###############################################################################
LICENSING_BRANCH?=$(PIN_BRANCH)
LICENSING_REPO?=github.com/tigera/licensing
LIBCALICO_REPO=github.com/tigera/libcalico-go-private

update-licensing-pin:
	$(call update_pin,github.com/tigera/licensing,$(LICENSING_REPO),$(LICENSING_BRANCH))

update-pins: update-licensing-pin replace-libcalico-pin

###############################################################################
# Building the binary
###############################################################################
BIN=bin/$(ARCH)
build: $(BIN)/install $(BIN)/calico $(BIN)/calico-ipam
ifeq ($(ARCH),amd64)
# Go only supports amd64 for Windows builds.
BIN_WIN=bin/windows
build: $(BIN_WIN)/calico.exe $(BIN_WIN)/calico-ipam.exe
endif
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*
	
## Build the Calico network plugin and ipam plugins
$(BIN)/install binary: $(LOCAL_BUILD_DEP) $(SRC_FILES)
	-mkdir -p .go-pkg-cache
	-mkdir -p $(BIN)
	$(DOCKER_RUN) \
	-e ARCH=$(ARCH) \
	-e GOARCH=$(ARCH) \
	-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	-v $(CURDIR)/$(BIN):/go/src/$(PACKAGE_NAME)/$(BIN):rw \
	$(LOCAL_BUILD_MOUNTS) \
	-w /go/src/$(PACKAGE_NAME) \
	-e GOCACHE=/go-cache \
	    $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $(BIN)/install -ldflags "-X main.VERSION=$(GIT_VERSION) -s -w" $(PACKAGE_NAME)/cmd/calico'

## Build the Calico network plugin and ipam plugins for Windows
$(BIN_WIN)/calico.exe $(BIN_WIN)/calico-ipam.exe: $(LOCAL_BUILD_DEP) $(SRC_FILES)
	$(DOCKER_RUN) \
	-e GOOS=windows \
	    $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go build -v -o $(BIN_WIN)/calico.exe -ldflags "-X main.VERSION=$(GIT_VERSION) -s -w" $(PACKAGE_NAME)/cmd/calico && \
		go build -v -o $(BIN_WIN)/calico-ipam.exe -ldflags "-X main.VERSION=$(GIT_VERSION) -s -w" $(PACKAGE_NAME)/cmd/calico'


###############################################################################
# Building the image
###############################################################################
image: $(DEPLOY_CONTAINER_MARKER)
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

$(DEPLOY_CONTAINER_MARKER): Dockerfile.$(ARCH) build fetch-cni-bins
	GO111MODULE=on docker build -t $(BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg GIT_VERSION=$(GIT_VERSION) -f Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	# Need amd64 builds tagged as :latest because Semaphore depends on that
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif
	touch $@

.PHONY: fetch-cni-bins
fetch-cni-bins: $(BIN)/flannel $(BIN)/loopback $(BIN)/host-local $(BIN)/portmap $(BIN)/tuning $(BIN)/bandwidth

$(BIN)/flannel $(BIN)/loopback $(BIN)/host-local $(BIN)/portmap $(BIN)/tuning $(BIN)/bandwidth:
	mkdir -p $(BIN)
	$(CURL) -L --retry 5 https://github.com/containernetworking/plugins/releases/download/$(CNI_VERSION)/cni-plugins-linux-$(ARCH)-$(CNI_VERSION).tgz | tar -xz -C $(BIN) ./flannel ./loopback ./host-local ./portmap ./tuning ./bandwidth

###############################################################################
# Static checks
###############################################################################
hooks_installed:=$(shell ./install-git-hooks)

###############################################################################
# Unit Tests
###############################################################################
## Run the unit tests.
ut: run-k8s-controller $(BIN)/install $(BIN)/host-local $(BIN)/calico-ipam $(BIN)/calico
	$(MAKE) ut-datastore DATASTORE_TYPE=etcdv3
	$(MAKE) ut-datastore DATASTORE_TYPE=kubernetes

$(BIN)/calico-ipam $(BIN)/calico: $(BIN)/install
	cp "$<" "$@"

ut-datastore: $(LOCAL_BUILD_DEP)
	# The tests need to run as root
	docker run --rm -t --privileged --net=host \
	$(EXTRA_DOCKER_ARGS) \
	-e ETCD_IP=$(LOCAL_IP_ENV) \
	-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	-e RUN_AS_ROOT=true \
	-e ARCH=$(ARCH) \
	-e PLUGIN=calico \
	-e BIN=/go/src/$(PACKAGE_NAME)/$(BIN) \
	-e CNI_SPEC_VERSION=$(CNI_SPEC_VERSION) \
	-e DATASTORE_TYPE=$(DATASTORE_TYPE) \
	-e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 \
	-e K8S_API_ENDPOINT=http://127.0.0.1:8080 \
	-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
			cd  /go/src/$(PACKAGE_NAME) && \
			ginkgo -cover -r -skipPackage pkg/install $(GINKGO_ARGS)'

ut-etcd: run-k8s-controller build $(BIN)/host-local
	$(MAKE) ut-datastore DATASTORE_TYPE=etcdv3
	make stop-etcd
	make stop-k8s-controller

ut-kdd: run-k8s-controller build $(BIN)/host-local
	$(MAKE) ut-datastore DATASTORE_TYPE=kubernetes
	make stop-etcd
	make stop-k8s-controller

## Run the tests in a container (as root) for different CNI spec versions
## to make sure we don't break backwards compatibility.
.PHONY: test-cni-versions
test-cni-versions:
	for cniversion in "0.2.0" "0.3.1" ; do \
		make ut CNI_SPEC_VERSION=$$cniversion; \
	done

config/crd: mod-download
	mkdir -p config/crd
	$(DOCKER_GO_BUILD) sh -c ' \
		cp -r `go list -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/config/crd/* config/crd; \
		chmod +w config/crd/*'

## Kubernetes apiserver used for tests
run-k8s-apiserver: config/crd stop-k8s-apiserver run-etcd
	docker run --detach --net=host \
	  --name calico-k8s-apiserver \
	  -v `pwd`/config:/config \
	  -v `pwd`/internal/pkg/testutils/private.key:/private.key \
	  gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION) kube-apiserver \
	    --etcd-servers=http://$(LOCAL_IP_ENV):2379 \
	    --service-cluster-ip-range=10.101.0.0/16 \
	    --service-account-key-file=/private.key
	# Wait until the apiserver is accepting requests.
	while ! docker exec calico-k8s-apiserver kubectl get nodes; do echo "Waiting for apiserver to come up..."; sleep 2; done
	while ! docker exec calico-k8s-apiserver kubectl apply -f /config/crd; do echo "Waiting for CRDs..."; sleep 2; done

## Kubernetes controller manager used for tests
run-k8s-controller: stop-k8s-controller run-k8s-apiserver
	docker run --detach --net=host \
	  --name calico-k8s-controller \
	  -v `pwd`/internal/pkg/testutils/private.key:/private.key \
	  gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION) kube-controller-manager \
	    --master=127.0.0.1:8080 \
	    --min-resync-period=3m \
	    --allocate-node-cidrs=true \
	    --cluster-cidr=192.168.0.0/16 \
	    --v=5 \
	    --service-account-private-key-file=/private.key

## Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f calico-k8s-apiserver

## Stop Kubernetes controller manager
stop-k8s-controller:
	@-docker rm -f calico-k8s-controller

## Etcd is used by the tests
run-etcd: stop-etcd
	docker run --detach \
	  -p 2379:2379 \
	  --name calico-etcd $(ETCD_CONTAINER) \
	  etcd \
	  --advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	  --listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stops calico-etcd containers
stop-etcd:
	@-docker rm -f calico-etcd

###############################################################################
# Install test
###############################################################################
# We pre-build the test binary so that we can run it outside a container and allow it
# to interact with docker.
pkg/install/install.test: pkg/install/*.go
	-mkdir -p .go-pkg-cache
	$(DOCKER_RUN) \
	-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	-v $(CURDIR)/.go-pkg-cache:/go/pkg/:rw \
		$(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
			cd /go/src/$(PACKAGE_NAME) && \
			go test ./pkg/install -c --tags install_test -o ./pkg/install/install.test'

.PHONY: test-install-cni
## Test the install
test-install-cni: image pkg/install/install.test
	cd pkg/install && CONTAINER_NAME=$(BUILD_IMAGE) ./install.test

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
ci: clean mod-download build static-checks test-cni-versions image-all test-install-cni

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
cd: check-dirty image-all cd-common

## Build fv binary for Windows
$(BIN_WIN)/win-fv.exe: $(LOCAL_BUILD_DEP) $(WINFV_SRCFILES)
	$(DOCKER_RUN) -e GOOS=windows $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go test ./win_tests -c -o $(BIN_WIN)/win-fv.exe'

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
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=$(VERSION)
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=latest

	# Copy artifacts for upload to GitHub.
	mkdir -p bin/github
	$(foreach var,$(VALIDARCHES), cp bin/$(var)/calico bin/github/calico-$(var);)
	$(foreach var,$(VALIDARCHES), cp bin/$(var)/calico-ipam bin/github/calico-ipam-$(var);)

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	docker run --rm $(BUILD_IMAGE):$(VERSION)-$(ARCH) calico -v | grep -x $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE):$(VERSION)-$(ARCH) calico -v` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm $(BUILD_IMAGE):$(VERSION)-$(ARCH) calico-ipam -v | grep -x $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE):$(VERSION)-$(ARCH) calico-ipam -v | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )

	# TODO: Some sort of quick validation of the produced binaries.

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
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=$(VERSION)

	# Push binaries to GitHub release.
	# Requires ghr: https://github.com/tcnksm/ghr
	# Requires GITHUB_TOKEN environment variable set.
	ghr -u tigera -r cni-plugin-private \
		-b "Release notes can be found at https://docs.projectcalico.org" \
		-n $(VERSION) \
		$(VERSION) ./bin/github/

	@echo "Confirm that the release was published at the following URL."
	@echo ""
	@echo "  https://github.com/tigera/cni-plugin/releases/tag/$(VERSION)"
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
	if ! docker run $(BUILD_IMAGE):latest-$(ARCH) calico -v | grep '^$(VERSION)$$'; then echo "Reported version:" `docker run $(BUILD_IMAGE):latest-$(ARCH) calico -v` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=latest

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN must be set for a release)
endif
ifeq (, $(shell which ghr))
	$(error Unable to find `ghr` in PATH, run this: go get -u github.com/tcnksm/ghr)
endif

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
## Run kube-proxy
run-kube-proxy:
	-docker rm -f calico-kube-proxy
	docker run --name calico-kube-proxy -d --net=host --privileged gcr.io/google_containers/hyperkube:$(K8S_VERSION) kube-proxy --master=http://127.0.0.1:8080 --v=2

.PHONY: test-watch
## Run the unit tests, watching for changes.
test-watch: $(BIN)/install run-etcd run-k8s-apiserver
	# The tests need to run as root
	CGO_ENABLED=0 ETCD_IP=127.0.0.1 PLUGIN=calico GOPATH=$(GOPATH) $(shell which ginkgo) watch -skipPackage pkg/install
