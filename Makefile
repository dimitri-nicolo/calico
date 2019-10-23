###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst docker-image/Dockerfile.%,%,$(wildcard docker-image/Dockerfile.*))

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

# Targets used when cross building.
.PHONY: register
# Enable binfmt adding support for miscellaneous binary formats.
# This is only needed when running non-native binaries.
register:
ifneq ($(BUILDARCH),$(ARCH))
	docker run --rm --privileged multiarch/qemu-user-static:register || true
endif

# list of arches *not* to build when doing *-all
#    until s390x works correctly
EXCLUDEARCH ?= s390x
VALIDARCHES = $(filter-out $(EXCLUDEARCH),$(ARCHES))


###############################################################################
CONTAINER_NAME=gcr.io/unique-caldron-775/cnx/tigera/cnx-apiserver
PACKAGE_NAME?=github.com/tigera/calico-k8sapiserver

GO_BUILD_VER?=v0.25
# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
CALICO_BUILD=calico/go-build:$(GO_BUILD_VER)


#This is a version with known container with compatible versions of sed/grep etc.
TOOLING_BUILD?=calico/go-build:v0.25



help:
	@echo "Calico K8sapiserver Makefile"
	@echo "Builds:"
	@echo
	@echo "  make all                   Build all the binary packages."
	@echo "  make tigera/cnx-apiserver  Build tigera/cnx-apiserver docker image."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make test                Run Tests."
	@echo "  sudo make kubeadm        Run a kubeadm master with the apiserver."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make update-vendor  Update the vendor directory with new "
	@echo "                      versions of upstream packages.  Record results"
	@echo "                      in glide.lock."
	@echo "  make clean         Remove binary files."
# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

all: tigera/cnx-apiserver
test: ut fv fv-kdd

# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Define some constants
#######################
K8S_VERSION    = v1.10.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg cmd
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
K8SAPISERVER_GO_FILES = $(shell find $(SRC_DIRS) -name \*.go -exec $(STAT) {} \; \
                   | sort -r | head -n 1 | sed "s/.* //")
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

CALICOAPISERVER_VERSION?=$(shell git describe --tags --dirty --always)
CALICOAPISERVER_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CALICOAPISERVER_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CALICOAPISERVER_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X $(PACKAGE_NAME)/cmd/apiserver/server.VERSION=$(CALICOAPISERVER_VERSION) \
	-X $(PACKAGE_NAME)/cmd/apiserver/server.BUILD_DATE=$(CALICOAPISERVER_BUILD_DATE) \
	-X $(PACKAGE_NAME)/cmd/apiserver/server.GIT_DESCRIPTION=$(CALICOAPISERVER_GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/cmd/apiserver/server.GIT_REVISION=$(CALICOAPISERVER_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"
KUBECONFIG_DIR?=/etc/kubernetes/admin.conf

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

# Volume-mount gopath into the build container to cache go module's packages. If the environment is using multiple
# comma-separated directories for gopath, use the first one, as that is the default one used by go modules.
ifneq ($(GOPATH),)
	# If the environment is using multiple comma-separated directories for gopath, use the first one, as that
	# is the default one used by go modules.
	GOMOD_CACHE = $(shell echo $(GOPATH) | cut -d':' -f1)/pkg/mod
else
	# If gopath is empty, default to $(HOME)/go.
	GOMOD_CACHE = $$HOME/go/pkg/mod
endif

EXTRA_DOCKER_ARGS += -v $(GOMOD_CACHE):/go/pkg/mod:rw

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

DOCKER_RUN := mkdir -p .go-pkg-cache .lint-cache $(GOMOD_CACHE) && \
			  docker run --rm \
				 --net=host \
				 $(EXTRA_DOCKER_ARGS) \
				 -e LOCAL_USER_ID=$(MY_UID) \
				 -e GOCACHE=/go-cache \
				 -e GOLANGCI_LINT_CACHE=/lint-cache \
				 -e GO111MODULE=off \
				 -v $(HOME)/.glide:/home/user/.glide:rw \
				 -v $${PWD}:/go/src/github.com/tigera/calico-k8sapiserver:rw \
				 -v $${PWD}/.go-pkg-cache:/go-cache:rw \
				 -v $${PWD}/.lint-cache:/lint-cache:rw \
				 -v $${PWD}/hack/boilerplate:/go/src/k8s.io/kubernetes/hack/boilerplate:rw \
				 -w /go/src/github.com/tigera/calico-k8sapiserver \
				 -e GOARCH=$(ARCH)

# Update the vendored dependencies with the latest upstream versions matching
# our glide.yaml.  If there area any changes, this updates glide.lock
# as a side effect.  Unless you're adding/updating a dependency, you probably
# want to use the vendor target to install the versions from glide.lock.
.PHONY: update-vendor
update-vendor:
	mkdir -p $$HOME/.glide
	$(DOCKER_RUN) $(CALICO_BUILD) glide up --strip-vendor
	touch vendor/.up-to-date

# vendor is a shortcut for force rebuilding the go vendor directory.
.PHONY: vendor
vendor vendor/.up-to-date: glide.lock
	mkdir -p $$HOME/.glide
	$(DOCKER_RUN) $(CALICO_BUILD) glide install --strip-vendor
	touch vendor/.up-to-date



###############################################################################
# Managing the upstream library pins
#
# If you're updating the pins with a non-release branch checked out,
# set PIN_BRANCH to the parent branch, e.g.:
#
#     PIN_BRANCH=release-v2.5 make update-pins
#        - or -
#     PIN_BRANCH=master make update-pins
#
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-libcalico-pin update-licensing-pin update-vendor

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

###############################################################################
## Set the default upstream repo branch to the current repo's branch,
## e.g. "master" or "release-vX.Y", but allow it to be overridden.
PIN_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)

###############################################################################
## libcalico

## Set the default LIBCALICO source for this project
LIBCALICO_PROJECT_DEFAULT=tigera/libcalico-go-private.git
LIBCALICO_GLIDE_LABEL=projectcalico/libcalico-go

LIBCALICO_BRANCH?=$(PIN_BRANCH)
LIBCALICO_REPO?=github.com/$(LIBCALICO_PROJECT_DEFAULT)
LIBCALICO_VERSION?=$(shell git ls-remote git@github.com:$(LIBCALICO_PROJECT_DEFAULT) $(LIBCALICO_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure LIBCALICO repo and branch are reachable
guard-git-libcalico:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Ensure the branch exists, or set LIBCALICO_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LIBCALICO_PROJECT_DEFAULT) "$(LIBCALICO_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(LIBCALICO_VERSION))" = "" ]; then \
		echo "ERROR: LIBCALICO version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-libcalico-pin: guard-ssh-forwarding-bug guard-git-libcalico
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(LIBCALICO_GLIDE_LABEL)" \
		REPO="$(LIBCALICO_REPO)" \
		VERSION="$(LIBCALICO_VERSION)" \
		DEFAULT_REPO="$(LIBCALICO_PROJECT_DEFAULT)" \
		BRANCH="$(LIBCALICO_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '



###############################################################################
## licensing

## Set the default LICENSING source for this project
LICENSING_PROJECT_DEFAULT=tigera/licensing
LICENSING_GLIDE_LABEL=tigera/licensing

LICENSING_BRANCH?=$(PIN_BRANCH)
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




# This section contains the code generation stuff
#################################################
.generate_exes: $(BINDIR)/defaulter-gen \
                $(BINDIR)/deepcopy-gen \
                $(BINDIR)/conversion-gen \
                $(BINDIR)/client-gen \
                $(BINDIR)/lister-gen \
                $(BINDIR)/informer-gen \
                $(BINDIR)/openapi-gen
	touch $@

$(BINDIR)/defaulter-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/defaulter-gen'

$(BINDIR)/deepcopy-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/deepcopy-gen'

$(BINDIR)/conversion-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/conversion-gen'

$(BINDIR)/client-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/client-gen'

$(BINDIR)/lister-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/lister-gen'

$(BINDIR)/informer-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/informer-gen'

$(BINDIR)/openapi-gen: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c 'go build -o $@ $(PACKAGE_NAME)/vendor/k8s.io/code-generator/cmd/openapi-gen'

# Regenerate all files if the gen exes changed or any "types.go" files changed
.generate_files: .generate_exes $(TYPES_FILES)
	# Generate defaults
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(BINDIR)/defaulter-gen \
		--v 1 --logtostderr \
		--go-header-file "/go/src/$(PACKAGE_NAME)/hack/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico/v3" \
		--extra-peer-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico" \
		--extra-peer-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico/v3" \
		--output-file-base "zz_generated.defaults"'
	# Generate deep copies
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(BINDIR)/deepcopy-gen \
		--v 1 --logtostderr \
		--go-header-file "/go/src/$(PACKAGE_NAME)/hack/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico/v3" \
		--bounding-dirs "github.com/tigera/calico-k8sapiserver" \
		--output-file-base zz_generated.deepcopy'
	# Generate conversions
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(BINDIR)/conversion-gen \
		--v 1 --logtostderr \
		--go-header-file "/go/src/$(PACKAGE_NAME)/hack/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico/v3" \
		--output-file-base zz_generated.conversion'
	# generate all pkg/client contents
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(BUILD_DIR)/update-client-gen.sh'
	# generate openapi
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(BINDIR)/openapi-gen \
		--v 1 --logtostderr \
		--go-header-file "/go/src/$(PACKAGE_NAME)/hack/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(PACKAGE_NAME)/pkg/apis/projectcalico/v3,k8s.io/api/core/v1,k8s.io/api/networking/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/util/intstr,github.com/projectcalico/libcalico-go/lib/apis/v3,github.com/projectcalico/libcalico-go/lib/apis/v1,github.com/projectcalico/libcalico-go/lib/numorstring" \
		--output-package "$(PACKAGE_NAME)/pkg/openapi"'
	touch $@

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

tag-image: imagetag tigera/cnx-apiserver
	docker tag tigera/cnx-apiserver:latest $(CONTAINER_NAME):$(IMAGETAG)

push-image: imagetag tag-image
	docker push $(CONTAINER_NAME):$(IMAGETAG)

###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
## Perform static checks on the code.
# TODO: re-enable these linters !
LINT_ARGS := --disable gosimple,govet,structcheck,errcheck,goimports,unused,ineffassign,staticcheck,deadcode,typecheck

static-checks: vendor
	$(DOCKER_RUN) \
	  $(CALICO_BUILD) \
	  golangci-lint run --deadline 5m $(LINT_ARGS)

# Run go fmt on all our go files.
# .PHONY: go-fmt goimports fix
fix go-fmt goimports:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'glide nv -x | \
	  grep -v -e "^\\.$$" | \
	  xargs goimports -w -local github.com/projectcalico/'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean static-checks tigera/cnx-apiserver fv ut check-generated-files

## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) push-image IMAGETAG=${BRANCH_NAME}

## Check if generated files are out of date
.PHONY: check-generated-files
check-generated-files: clean-generated .generate_files
	if (git describe --tags --dirty | grep -c dirty >/dev/null); then \
	  echo "Generated files are out of date."; \
	  false; \
	else \
	  echo "Generated files are up to date."; \
	fi

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "apiserver" instead of "$(BINDIR)/apiserver".
#########################################################################
$(BINDIR)/calico-k8sapiserver: vendor/.up-to-date .generate_files $(K8SAPISERVER_GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo Building k8sapiserver...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) \
		sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/apiserver" && \
		( ldd $(BINDIR)/calico-k8sapiserver 2>&1 | grep -q "Not a valid dynamic program" || \
		( echo "Error: $(BINDIR)/calico-k8sapiserver was not statically linked"; false ) )'

# Build the tigera/cnx-apiserver docker image.
.PHONY: tigera/cnx-apiserver
tigera/cnx-apiserver: vendor/.up-to-date .generate_files $(BINDIR)/calico-k8sapiserver
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp $(BINDIR)/calico-k8sapiserver docker-image/bin/
	docker build --pull -t tigera/cnx-apiserver --file ./docker-image/Dockerfile.$(ARCH) docker-image

.PHONY: ut
ut: vendor/.up-to-date run-etcd
	$(DOCKER_RUN) $(CALICO_BUILD) \
		sh -c 'ETCD_ENDPOINTS="http://127.0.0.1:2379" DATASTORE_TYPE="etcdv3" go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

## Run etcd as a container (calico-etcd)
run-etcd: stop-etcd
	docker run --detach \
	--net=host \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:v3.1.7 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stop the etcd container (calico-etcd)
stop-etcd:
	-docker rm -f calico-etcd

.PHONY: fv
fv: vendor/.up-to-date run-etcd
	$(DOCKER_RUN) $(CALICO_BUILD) \
		sh -c 'ETCD_ENDPOINTS="http://127.0.0.1:2379" DATASTORE_TYPE="etcdv3" test/integration.sh'

## Run a local kubernetes master with API via hyperkube
run-kubernetes-master: run-etcd stop-kubernetes-master
	# Run a Kubernetes apiserver using Docker.
	docker run \
		--net=host --name st-apiserver \
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--logtostderr=true

	# Wait until we can configure a cluster role binding which allows anonymous auth.
	while ! docker exec st-apiserver kubectl create clusterrolebinding anonymous-admin --clusterrole=cluster-admin --user=system:anonymous; do echo "Trying to create ClusterRoleBinding"; sleep 2; done

	# And run the controller manager.
	docker run \
		--net=host --name st-controller-manager \
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube controller-manager \
			--master=127.0.0.1:8080 \
			--min-resync-period=3m \
			--allocate-node-cidrs=true \
			--cluster-cidr=10.10.0.0/16 \
			--v=5

	# Create CustomResourceDefinition (CRD) for Calico resources
	# from the manifest crds.yaml
	docker run \
		--net=host \
		--rm \
		-v  $(CURDIR)/vendor/github.com/projectcalico/libcalico-go:/manifests \
		lachlanevenson/k8s-kubectl:${K8S_VERSION} \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/test/crds.yaml

	# Create a Node in the API for the tests to use.
	docker run \
		--net=host \
		--rm \
		-v  $(CURDIR)/vendor/github.com/projectcalico/libcalico-go:/manifests \
		lachlanevenson/k8s-kubectl:${K8S_VERSION} \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/test/mock-node.yaml

	# Create Namespaces required by namespaced Calico `NetworkPolicy`
	# tests from the manifests namespaces.yaml.
	docker run \
		--net=host \
		--rm \
		-v  $(CURDIR)/vendor/github.com/projectcalico/libcalico-go:/manifests \
		lachlanevenson/k8s-kubectl:${K8S_VERSION} \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/test/namespaces.yaml

## Stop the local kubernetes master
stop-kubernetes-master:
	# Delete the cluster role binding.
	-docker exec st-apiserver kubectl delete clusterrolebinding anonymous-admin

	# Stop master components.
	-docker rm -f st-apiserver st-controller-manager

.PHONY: fv-kdd
fv-kdd: vendor/.up-to-date run-kubernetes-master
	$(DOCKER_RUN) $(CALICO_BUILD) \
		sh -c 'K8S_API_ENDPOINT="http://127.0.0.1:8080" DATASTORE_TYPE="kubernetes" test/integration.sh'

.PHONY: clean
clean: clean-bin clean-build-image clean-generated
clean-build-image:
	docker rmi -f tigera/cnx-apiserver > /dev/null 2>&1 || true

clean-generated:
	rm -f .generate_files
	find $(TOP_SRC_DIRS) -name zz_generated* -exec rm {} \;
	# rollback changes to the generated clientset directories
	# find $(TOP_SRC_DIRS) -type d -name *_generated -exec rm -rf {} \;

clean-bin:
	rm -rf $(BINDIR) \
			.generate_exes \
			docker-image/bin

.PHONY: release
release: clean
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif

	# Rebuild all the checked in generated files.  If any weren't the same, then
	# the dirty check will fail.
	$(MAKE) .generate_files

	git tag $(VERSION)

	# Check to make sure the tag isn't "dirty"
	if git describe --tags --dirty | grep dirty; \
	then echo current git working tree is "dirty". Make sure you do not have any uncommitted changes ;false; fi

	# Build the apiserver binaries and image
	$(MAKE) tigera/cnx-apiserver

	# Check that the version output includes the version specified.
	# Tests that the "git tag" makes it into the binaries. Main point is to catch "-dirty" builds
	# Release is currently supported on darwin / linux only.
	if ! docker run tigera/cnx-apiserver | grep 'Version:\s*$(VERSION)$$'; then \
	  echo "Reported version:" `docker run tigera/cnx-apiserver` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

	# Retag images with correct version and GCR private registry
	docker tag tigera/cnx-apiserver $(CONTAINER_NAME):$(VERSION)

	# Check that images were created recently and that the IDs of the versioned and latest images match
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" tigera/cnx-apiserver
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(CONTAINER_NAME):$(VERSION)

	@echo "\nNow push the tag and images. Then create a release on Github and"
	@echo "\nAdd release notes for calico-k8sapiserver. Use this command"
	@echo "to find commit messages for this release: git log --oneline <old_release_version>...$(VERSION)"
	@echo "git push origin $(VERSION)"
	@echo "gcloud auth configure-docker"
	@echo "docker push $(CONTAINER_NAME):$(VERSION)"

.PHONY: kubeadm
kubeadm:
	kubeadm reset
	rm -rf /var/etcd
	kubeadm init --config artifacts/misc/kubeadm.yaml

	# Wait for it to be ready
	while ! KUBECONFIG=$(KUBECONFIG_DIR) kubectl get pods; do sleep 15; done

	# Install Calico and the AAPI server
	KUBECONFIG=$(KUBECONFIG_DIR) kubectl apply -f artifacts/misc/calico.yaml
	KUBECONFIG=$(KUBECONFIG_DIR) kubectl taint nodes --all node-role.kubernetes.io/master-
	KUBECONFIG=$(KUBECONFIG_DIR) kubectl create namespace calico
	KUBECONFIG=$(KUBECONFIG_DIR) kubectl create -f artifacts/example/
	@echo "Kubeadm master created."
	@echo "To use, run the following commands:"
	@echo "sudo cp $(KUBECONFIG_DIR) \$$HOME"
	@echo "sudo chown \$$(id -u):\$$(id -g) \$$HOME/admin.conf"
	@echo "export KUBECONFIG=\$$HOME/admin.conf"
	@echo "kubectl get tiers"

# Run fossa.io license checks
foss-checks: vendor
	@echo Running $@...
	@docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -e LOCAL_USER_ID=$(MY_UID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(CALICO_BUILD) /usr/local/bin/fossa
