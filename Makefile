# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut test-kdd test-etcd

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

BIN=bin/$(ARCH)

GO_BUILD_VER?=v0.26
CALICO_BUILD = calico/go-build:$(GO_BUILD_VER)
PACKAGE_NAME ?= github.com/projectcalico/confd

CALICOCTL_VER=master
CALICOCTL_CONTAINER_NAME=gcr.io/unique-caldron-775/cnx/tigera/calicoctl:$(CALICOCTL_VER)-$(ARCH)
BIRD_VER=v0.3.3-138-ge37e4770
BIRD_CONTAINER_NAME=calico/bird:$(BIRD_VER)-$(ARCH)
TYPHA_VER=master
TYPHA_CONTAINER_NAME=gcr.io/unique-caldron-775/cnx/tigera/typha:$(TYPHA_VER)-$(ARCH)
K8S_VERSION?=v1.14.1
ETCD_VER?=v3.3.7
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')

GIT_DESCRIPTION:=$(shell git describe --tags || echo '<unknown>')
LDFLAGS=-ldflags "-X $(PACKAGE_NAME)/pkg/buildinfo.GitVersion=$(GIT_DESCRIPTION)"

# Figure out the users UID.  This is needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
LOCAL_USER_ID?=$(shell id -u $$USER)

EXTRA_DOCKER_ARGS	+= -e GO111MODULE=on -e GOPRIVATE=github.com/tigera/*
GIT_CONFIG_SSH		?= git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"

# All go files.
SRC_FILES:=$(shell find . -name '*.go' -not -path "./vendor/*" )

# Allow the ssh auth sock to be mapped into the build container.
ifdef SSH_AUTH_SOCK
	EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

# Volume-mount gopath into the build container to cache go module's packages. If the environment is using multiple
# comma-separated directories for gopath, use the first one, as that is the default one used by go modules.
ifneq ($(GOPATH),)
	# If the environment is using multiple comma-separated directories for gopath, use the first one, as that
	# is the default one used by go modules.
	GOMOD_CACHE = $(shell echo $(GOPATH) | cut -d':' -f1)/pkg/mod
else
	# If gopath is empty, default to $(HOME)/go.
	GOMOD_CACHE = $(HOME)/go/pkg/mod
endif

EXTRA_DOCKER_ARGS	+= -v $(GOMOD_CACHE):/go/pkg/mod:rw

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
PHONY:local_build

ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go:rw \
	-v $(CURDIR)/../typha-private:/go/src/github.com/projectcalico/typha:rw
local_build:
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
	$(DOCKER_RUN) $(CALICO_BUILD) go mod edit -replace=github.com/projectcalico/typha=../typha
else
local_build:
	@echo "Building confd-private"
endif

DOCKER_RUN := mkdir -p .go-pkg-cache $(GOMOD_CACHE) $(BIN) tests/logs && \
	docker run --rm \
		--net=host \
		$(EXTRA_DOCKER_ARGS) \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-e GOCACHE=/go-cache \
		-e GOARCH=$(ARCH) \
		-e GOPATH=/go \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-v $(CURDIR)/.go-pkg-cache:/go-cache:rw \
		-w /go/src/$(PACKAGE_NAME)

.PHONY: clean
clean:
	rm -rf vendor
	rm -rf bin/*
	rm -rf tests/logs

.PHONY: install-git-hooks
install-git-hooks:
	./install-git-hooks

###############################################################################
# Building the binary
###############################################################################
build: local_build bin/confd
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/confd-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) && \
		go build -v -i -o $@ $(LDFLAGS) "$(PACKAGE_NAME)" && \
		( ldd bin/confd-$(ARCH) 2>&1 | grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
		( echo "Error: bin/confd was not statically linked"; false ) )'

bin/confd: bin/confd-$(ARCH)
ifeq ($(ARCH),amd64)
	ln -f bin/confd-$(ARCH) bin/confd
endif

# Cross-compile confd for Windows
windows-packaging/tigera-confd.exe: $(SRC_FILES)
	@echo Building confd for Windows...
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) && \
		GOOS=windows go build -v -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)" && \
		( ldd $@ 2>&1 | grep -q "Not a valid dynamic program" || \
		( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# Updating pins
###############################################################################
PIN_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)

define get_remote_version
	$(shell git ls-remote ssh://git@$(1) $(2) 2>/dev/null | cut -f 1)
endef

# update_pin updates the given package's version to the latest available in the specified repo and branch.
# $(1) should be the name of the package, $(2) and $(3) the repository and branch from which to update it.
define update_pin
	$(eval new_ver := $(call get_remote_version,$(2),$(3)))

	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH); \
		if [[ ! -z "$(new_ver)" ]]; then \
			go get $(1)@$(new_ver); \
			go mod download; \
		fi'
endef

# update_replace_pin updates the given package's version to the latest available in the specified repo and branch.
# This routine can only be used for packages being replaced in go.mod, such as private versions of open-source packages.
# $(1) should be the name of the package, $(2) and $(3) the repository and branch from which to update it.
define update_replace_pin
	$(eval new_ver := $(call get_remote_version,$(2),$(3)))
	$(DOCKER_RUN) -i $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH); \
		if [[ ! -z "$(new_ver)" ]]; then \
			go mod edit -replace $(1)=$(2)@$(new_ver); \
			go mod download; \
		fi'
endef

guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

LIBCALICO_BRANCH?=$(PIN_BRANCH)
LIBCALICO_REPO?=github.com/tigera/libcalico-go-private
LICENSING_BRANCH?=$(PIN_BRANCH)
LICENSING_REPO?=github.com/tigera/licensing
TYPHA_BRANCH?=$(PIN_BRANCH)
TYPHA_REPO?=github.com/tigera/typha-private

update-libcalico-pin: guard-ssh-forwarding-bug
	$(call update_replace_pin,github.com/projectcalico/libcalico-go,$(LIBCALICO_REPO),$(LIBCALICO_BRANCH))

update-licensing-pin: guard-ssh-forwarding-bug
	$(call update_pin,github.com/tigera/licensing,$(LICENSING_REPO),$(LICENSING_BRANCH))

update-typha-pin: guard-ssh-forwarding-bug
	$(call update_replace_pin,github.com/projectcalico/typha,$(TYPHA_REPO),$(TYPHA_BRANCH))

git-status:
	git status --porcelain

git-config:
ifdef CONFIRM
	git config --global user.name "Semaphore Automatic Update"
	git config --global user.email "marvin@tigera.io"
endif

git-commit:
	git diff --quiet HEAD || git commit -m "Semaphore Automatic Update" go.mod go.sum

git-push:
	git push

update-pins: update-libcalico-pin update-licensing-pin update-typha-pin

commit-pin-updates: update-pins git-status ci git-config git-commit git-push

###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
LINT_ARGS := --deadline 5m --max-issues-per-linter 0 --max-same-issues 0
static-checks:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH); golangci-lint run $(LINT_ARGS)'

.PHONY: fix
fix:
	goimports -w $(SRC_FILES)

.PHONY: foss-checks
foss-checks:
	$(DOCKER_RUN) -e FOSSA_API_KEY=$(FOSSA_API_KEY) $(CALICO_BUILD) /usr/local/bin/fossa

###############################################################################
# Unit Tests
###############################################################################

# Set to true when calling test-xxx to update the rendered templates instead of
# checking them.
UPDATE_EXPECTED_DATA?=false

.PHONY: test-kdd
## Run template tests against KDD
test-kdd: bin/confd bin/kubectl bin/bird bin/bird6 bin/calico-node bin/calicoctl bin/typha run-k8s-apiserver
	-git clean -fx etc/calico/confd
	docker run --rm \
	        $(EXTRA_DOCKER_ARGS) \
		-v $(CURDIR)/tests/:/tests/ \
		-v $(CURDIR)/bin:/calico/bin/ \
		-v $(CURDIR)/etc/calico:/etc/calico/ \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-e GOPATH=/go \
		-e LOCAL_USER_ID=0 \
		-e FELIX_TYPHAADDR=127.0.0.1:5473 \
		-e FELIX_TYPHAREADTIMEOUT=50 \
		-e UPDATE_EXPECTED_DATA=$(UPDATE_EXPECTED_DATA) \
		-e KUBECONFIG=/tests/confd_kubeconfig \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) /bin/bash -c '$(GIT_CONFIG_SSH); /tests/test_suite_kdd.sh || \
	{ \
	    echo; \
	    echo === confd single-shot log:; \
	    cat tests/logs/kdd/logss || true; \
	    echo; \
	    echo === confd daemon log:; \
	    cat tests/logs/kdd/logd1 || true; \
	    echo; \
	    echo === Typha log:; \
	    cat tests/logs/kdd/typha || true; \
	    echo; \
	    false; \
	}'
	-git clean -fx etc/calico/confd

.PHONY: test-etcd
## Run template tests against etcd
test-etcd: bin/confd bin/etcdctl bin/bird bin/bird6 bin/calico-node bin/kubectl bin/calicoctl run-etcd run-k8s-apiserver
	-git clean -fx etc/calico/confd
	docker run --rm \
		-v $(CURDIR)/tests/:/tests/ \
		-v $(CURDIR)/bin:/calico/bin/ \
		-v $(CURDIR)/etc/calico:/etc/calico/ \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-e GOPATH=/go \
		-e LOCAL_USER_ID=0 \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-e UPDATE_EXPECTED_DATA=$(UPDATE_EXPECTED_DATA) \
		-e GO111MODULE=on \
		-e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 \
		-e ETCDCTL_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 \
		-e KUBECONFIG=/tests/confd_kubeconfig \
		$(CALICO_BUILD) /bin/bash -c '$(GIT_CONFIG_SSH); /tests/test_suite_etcd.sh'
	-git clean -fx etc/calico/confd

.PHONY: ut
## Run the fast set of unit tests in a container.
ut: local_build
	$(DOCKER_RUN) --privileged $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) && cd /go/src/$(PACKAGE_NAME) && ginkgo -r .'

## Etcd is used by the kubernetes
# NOTE: https://quay.io/repository/coreos/etcd is available *only* for the following archs with the following tags:
# amd64: 3.2.5
# arm64: 3.2.5-arm64
# ppc64le: 3.2.5-ppc64le
# s390x is not available
COREOS_ETCD ?= quay.io/coreos/etcd:$(ETCD_VER)-$(ARCH)
ifeq ($(ARCH),amd64)
COREOS_ETCD = quay.io/coreos/etcd:$(ETCD_VER)
endif
run-etcd: stop-etcd
	docker run --detach \
	-e GO111MODULE=on \
	--net=host \
	--name calico-etcd $(COREOS_ETCD) \
	etcd \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://$(LOCAL_IP_ENV):4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stops calico-etcd containers
stop-etcd:
	@-docker rm -f calico-etcd

.PHONY: tests/confd_kubeconfig
tests/confd_kubeconfig: tests/confd_kubeconfig.in
	sed s/@@LOCAL_IP_ENV@@/$(LOCAL_IP_ENV)/ < tests/confd_kubeconfig.in > tests/confd_kubeconfig

## Kubernetes apiserver used for tests
run-k8s-apiserver: stop-k8s-apiserver run-etcd tests/confd_kubeconfig
	docker run --detach --net=host \
	  --name calico-k8s-apiserver \
	gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION) \
		  /hyperkube apiserver --etcd-servers=http://$(LOCAL_IP_ENV):2379 \
		  --service-cluster-ip-range=10.101.0.0/16 --insecure-bind-address=$(LOCAL_IP_ENV)
	# Wait until the apiserver is accepting requests.
	docker cp tests/confd_kubeconfig calico-k8s-apiserver:/kubeconfig
	while ! docker exec calico-k8s-apiserver kubectl --kubeconfig=/kubeconfig get nodes; do echo "Waiting for apiserver to come up..."; sleep 2; done

## Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f calico-k8s-apiserver

bin/kubectl:
	curl -sSf -L --retry 5 https://storage.googleapis.com/kubernetes-release/release/$(K8S_VERSION)/bin/linux/$(ARCH)/kubectl -o $@
	chmod +x $@

bin/bird bin/bird6:
	-docker rm -f calico-bird
	# Latest BIRD binaries are stored in automated builds of calico/bird.
	# To get them, we create (but don't start) a container from that image.
	docker pull $(BIRD_CONTAINER_NAME)
	docker create --name calico-bird $(BIRD_CONTAINER_NAME) /bin/sh
	# Then we copy the files out of the container.  Since docker preserves
	# mtimes on its copy, check the file really did appear, then touch it
	# to make sure that downstream targets get rebuilt.
	docker cp calico-bird:/bird bin/ && \
	docker cp calico-bird:/bird6 bin/ && \
	  test -e $@ && \
	  touch $@
	-docker rm -f calico-bird

bin/calico-node:
	cp fakebinary $@
	chmod +x $@

bin/etcdctl:
	curl -sSf -L --retry 5  https://github.com/coreos/etcd/releases/download/$(ETCD_VER)/etcd-$(ETCD_VER)-linux-$(ARCH).tar.gz | tar -xz -C bin --strip-components=1 etcd-$(ETCD_VER)-linux-$(ARCH)/etcdctl

bin/calicoctl:
	-docker rm -f calico-ctl
	# Latest calicoctl binaries are stored in automated builds of calico/ctl.
	# To get them, we create (but don't start) a container from that image.
	docker pull $(CALICOCTL_CONTAINER_NAME)
	docker create --name calico-ctl $(CALICOCTL_CONTAINER_NAME)
	# Then we copy the files out of the container.  Since docker preserves
	# mtimes on its copy, check the file really did appear, then touch it
	# to make sure that downstream targets get rebuilt.
	docker cp calico-ctl:/calicoctl $@ && \
	  test -e $@ && \
	  touch $@
	-docker rm -f calico-ctl

bin/typha:
	-docker rm -f confd-typha
	docker pull $(TYPHA_CONTAINER_NAME)
	docker create --name confd-typha $(TYPHA_CONTAINER_NAME)
	# Then we copy the files out of the container.  Since docker preserves
	# mtimes on its copy, check the file really did appear, then touch it
	# to make sure that downstream targets get rebuilt.
	docker cp confd-typha:/code/calico-typha $@ && \
	  test -e $@ && \
	  touch $@
	-docker rm -f confd-typha

###############################################################################
# CI
###############################################################################
.PHONY: mod-download
mod-download:
	-$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH); go mod download'

.PHONY: ci
ci: clean mod-download static-checks test

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-windows-archive

## Produces a git tag for the release.
release-tag: release-prereqs release-notes
	git tag $(VERSION) -F release-notes-$(VERSION)
	@echo ""
	@echo "Now you can publish the release:"
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish"
	@echo ""

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	@echo "Confirm that the release was published at the following URL."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif

# Files to include in the Windows ZIP archive.
WINDOWS_BUILT_FILES := windows-packaging/tigera-confd.exe
# Name of the Windows release ZIP archive.
WINDOWS_ARCHIVE := dist/tigera-confd-windows-$(VERSION).zip

## Produces the Windows ZIP archive for the release.
release-windows-archive $(WINDOWS_ARCHIVE): release-prereqs $(WINDOWS_BUILT_FILES)
	-rm -f $(WINDOWS_ARCHIVE)
	mkdir -p dist
	cd windows-packaging && zip -r ../$(WINDOWS_ARCHIVE) .

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
help:
	@echo "confd Makefile"
	@echo
	@echo "Dependencies: docker 1.12+; go 1.8+"
	@echo
	@echo "For any target, set ARCH=<target> to build for a given target."
	@echo "For example, to build for arm64:"
	@echo
	@echo "  make build ARCH=arm64"
	@echo
	@echo "Builds:"
	@echo
	@echo "  make build	Build the binary."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make test	Run all tests."
	@echo "  make test-kdd	Run kdd tests."
	@echo "  make test-etcd	Run etcd tests."
	@echo
	@echo "Maintenance:"
	@echo "  make clean	Remove binary files and docker images."
	@echo "-----------------------------------------"
	@echo "ARCH (target):	$(ARCH)"
	@echo "BUILDARCH (host):$(BUILDARCH)"
	@echo "CALICO_BUILD:	$(CALICO_BUILD)"
	@echo "-----------------------------------------"
