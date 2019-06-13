# This Makefile builds Felix and packages it in various forms:
#
#                                                                      Go install
#                                                                         Glide
#                                                                           |
#                                                                           |
#                                                                           |
#                                                    +-------+              v
#                                                    | Felix |   +---------------------+
#                                                    |  Go   |   | calico/go-build     |
#                                                    |  code |   +---------------------+
#                                                    +-------+         /
#                                                           \         /
#                                                            \       /
#                                                             \     /
#                                                             go build
#                                                                 \
#                                                                  \
#                                                                   \
# +----------------------+                                           :
# | calico-build/centos7 |                                           v
# | calico-build/xenial  |                                 +------------------+
# | calico-build/trusty  |                                 | bin/calico-felix |
# +----------------------+                                 +------------------+
#                     \                                          /   /
#                      \             .--------------------------'   /
#                       \           /                              /
#                        \         /                      .-------'
#                         \       /                      /
#                     rpm/build-rpms                    /
#                   debian/build-debs                  /
#                           |                         /
#                           |                   docker build
#                           v                         |
#            +----------------------------+           |
#            |  RPM packages for Centos7  |           |
#            |  RPM packages for Centos6  |           v
#            | Debian packages for Xenial |    +--------------+
#            | Debian packages for Trusty |    | tigera/felix |
#            +----------------------------+    +--------------+
#
#
#
###############################################################################
# Shortcut targets
default: build

## Build binary for current platform
all: build

## Run the tests for the current platform/architecture
test: ut fv

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

# Build mounts for running in "local build" mode. Mount in libcalico, but null out
# the vendor directory. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
LOCAL_BUILD_MOUNTS ?=
ifeq ($(LOCAL_BUILD),true)
LOCAL_BUILD_MOUNTS = -v $(CURDIR)/../libcalico-go:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/libcalico-go/vendor:ro \
	-v $(CURDIR)/../typha:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/typha:ro \
	-v $(CURDIR)/.empty:/go/src/$(PACKAGE_NAME)/vendor/github.com/projectcalico/typha/vendor:ro
endif

# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

# Targets used when cross building.
.PHONY: native register
native:
ifneq ($(BUILDARCH),$(ARCH))
	@echo "Target $(MAKECMDGOALS)" is not supported when cross building! && false
endif

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
BUILD_IMAGE?=tigera/felix
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=
PACKAGE_NAME?=github.com/projectcalico/felix

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

GO_BUILD_VER?=v0.20
# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
CALICO_BUILD=calico/go-build:$(GO_BUILD_VER)
ETCD_VERSION?=v3.3.7
K8S_VERSION?=v1.11.3
PROTOC_VER?=v0.1
PROTOC_CONTAINER ?=calico/protoc:$(PROTOC_VER)-$(BUILDARCH)

#This is a version with known container with compatible versions of sed/grep etc.
TOOLING_BUILD?=calico/go-build:v0.20


FV_ETCDIMAGE ?= quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
FV_K8SIMAGE ?= gcr.io/google_containers/hyperkube-$(BUILDARCH):$(K8S_VERSION)
FV_TYPHAIMAGE ?= tigera/typha:latest-$(BUILDARCH)
FV_FELIXIMAGE ?= tigera/felix:latest-$(BUILDARCH)

# If building on amd64 omit the arch in the container name.  Fixme!
ifeq ($(BUILDARCH),amd64)
        FV_ETCDIMAGE=quay.io/coreos/etcd:$(ETCD_VERSION)
        FV_K8SIMAGE=gcr.io/google_containers/hyperkube:$(K8S_VERSION)
        FV_TYPHAIMAGE=gcr.io/unique-caldron-775/cnx/tigera/typha:master
endif

# Total number of ginkgo batches to run.  The CI system sets this according to the number
# of jobs that it divides the FVs into.
FV_NUM_BATCHES?=3
# Space-delimited list of FV batches to run in parallel.  Defaults to running all batches
# in parallel on this host.  The CI system runs a subset of batches in each parallel job.
FV_BATCHES_TO_RUN?=$(shell seq $(FV_NUM_BATCHES))
FV_SLOW_SPEC_THRESH=90

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
GIT_COMMIT:=$(shell git rev-parse HEAD || echo '<unknown>')
GIT_SHORT_COMMIT:=$(shell git rev-parse --short HEAD || echo '<unknown>')
BUILD_ID:=$(shell git rev-parse HEAD || uuidgen | sed 's/-//g')
GIT_DESCRIPTION:=$(shell git describe --tags --dirty --always || echo '<unknown>')
ifeq ($(LOCAL_BUILD),true)
	GIT_DESCRIPTION = $(shell git describe --tags --dirty --always || echo '<unknown>')-dev-build
endif

# Calculate a timestamp for any build artefacts.
DATE:=$(shell date -u +'%FT%T%z')

# Linker flags for building Felix.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
        -X $(PACKAGE_NAME)/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
        -X $(PACKAGE_NAME)/buildinfo.BuildDate=$(DATE) \
        -X $(PACKAGE_NAME)/buildinfo.GitRevision=$(GIT_COMMIT) \
        -B 0x$(BUILD_ID)"

# List of Go files that are generated by the build process.  Builds should
# depend on these, clean removes them.
GENERATED_FILES:=proto/felixbackend.pb.go bpf/bpf-packr.go bpf/packrd/packed-packr.go bpf/xdp/generated/xdp.o

# All Felix go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_FELIX_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print) $(GENERATED_FILES)

# Files to include in the Windows ZIP archive.
WINDOWS_ARCHIVE_FILES := bin/tigera-felix.exe windows-packaging/README.txt windows-packaging/*.ps1
# Name of the Windows release ZIP archive.
WINDOWS_ARCHIVE := dist/tigera-felix-windows-$(VERSION).zip

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
LOCAL_USER_ID:=$(shell id -u)
LOCAL_GROUP_ID:=$(shell id -g)

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif
DOCKER_RUN := mkdir -p .go-pkg-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
                              -e GOCACHE=/gocache \
                              -v $(HOME)/.glide:/home/user/.glide:rw \
                              -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
                              -v $(CURDIR)/.go-pkg-cache:/gocache:rw \
                              -w /go/src/$(PACKAGE_NAME) \
                              -e GOARCH=$(ARCH)

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/bin \
	       dist \
	       build \
	       fv/fv.test \
	       $(GENERATED_FILES) \
	       docs/*.pdf \
	       .glide \
	       vendor \
	       .go-pkg-cache \
	       check-licenses/dependency-licenses.txt \
	       release-notes-*
	find . -name "junit.xml" -type f -delete
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

###############################################################################
# Building the binary
###############################################################################
build: bin/calico-felix
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

# Update the vendored dependencies with the latest upstream versions matching
# our glide.yaml.  If there area any changes, this updates glide.lock
# as a side effect.  Unless you're adding/updating a dependency, you probably
# want to use the vendor target to install the versions from glide.lock.
VENDOR_REMADE := false
.PHONY: update-vendor
update-vendor glide.lock:
	mkdir -p $$HOME/.glide
	$(DOCKER_RUN) $(CALICO_BUILD) glide up --strip-vendor
	touch vendor/.up-to-date
	# Optimization: since glide up does the job of glide install, flag to the
	# vendor target that it doesn't need to do anything.
	$(eval VENDOR_REMADE := true)

# vendor is a shortcut for force rebuilding the go vendor directory.
.PHONY: vendor
vendor vendor/.up-to-date: glide.lock
	if ! $(VENDOR_REMADE); then \
	  mkdir -p $$HOME/.glide && \
	  $(DOCKER_RUN) $(CALICO_BUILD) glide install --strip-vendor && \
	  touch vendor/.up-to-date; \
	fi

bin/calico-felix: bin/calico-felix-$(ARCH)
	ln -f bin/calico-felix-$(ARCH) bin/calico-felix

bin/calico-felix-$(ARCH): $(SRC_FILES) vendor/.up-to-date
	@echo Building felix for $(ARCH) on $(BUILDARCH)
	mkdir -p bin
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
	   sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-felix" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: $@ was not statically linked"; false ) )'

# Cross-compile Felix for Windows
bin/calico-felix.exe: $(SRC_FILES) vendor/.up-to-date
	@echo Building felix for Windows...
	mkdir -p bin
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
           sh -c 'GOOS=windows go build -v -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-felix" && \
		( ldd $@ 2>&1 | grep -q "Not a valid dynamic program" || \
		( echo "Error: $@ was not statically linked"; false ) )'

bin/tigera-felix.exe: bin/calico-felix.exe
	cp $< $@

%.url: % utils/make-azure-blob.sh
	utils/make-azure-blob.sh $< $(notdir $(basename $<))-$(GIT_SHORT_COMMIT)$(suffix $<) \
	    felix-test-uploads felixtestuploads felixtestuploads > $@.tmp
	mv $@.tmp $@

windows-felix-url: bin/tigera-felix.exe.url
	@echo
	@echo calico-felix.exe download link:
	@cat $<
	@echo
	@echo Powershell download command:
	@echo "Invoke-WebRequest '`cat $<`' -O tigera-felix-$(GIT_SHORT_COMMIT).exe"

windows-zip-url:
ifndef VERSION
	$(MAKE) windows-zip-url VERSION=dev
else
	$(MAKE) $(WINDOWS_ARCHIVE).url VERSION=dev
	@echo
	@echo $(WINDOWS_ARCHIVE) download link:
	@cat $(WINDOWS_ARCHIVE).url
	@echo
	@echo Powershell download command:
	@echo "Invoke-WebRequest '`cat $(WINDOWS_ARCHIVE).url`' -O tigera-felix.zip"
endif

# Generate the protobuf bindings for go. The proto/felixbackend.pb.go file is included in SRC_FILES
protobuf proto/felixbackend.pb.go: proto/felixbackend.proto
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
                  -v $(CURDIR):/code -v $(CURDIR)/proto:/src:rw \
	              $(PROTOC_CONTAINER) \
	              --gogofaster_out=plugins=grpc:. \
	              felixbackend.proto

bpf/xdp/generated/xdp.o: bpf/xdp/filter.c
	mkdir -p bpf/xdp/generated
	# the bpf object file is not arch dependent, so we can build with the current ARCH
	docker build -t calico-build/bpf-clang -f docker-build-images/bpf-clang-builder.Dockerfile.$(BUILDARCH) .
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
	          -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	              calico-build/bpf-clang \
	              /bin/sh -c \
	              "cd /go/src/$(PACKAGE_NAME) && \
	               clang \
	                      -D__KERNEL__ \
	                      -D__ASM_SYSREG_H \
	                      -Wno-unused-value \
	                      -Wno-pointer-sign \
	                      -Wno-compare-distinct-pointer-types \
	                      -Wunused \
	                      -Wall \
	                      -Werror \
	                      -fno-stack-protector \
	                      -O2 \
	                      -emit-llvm \
	                      -c /go/src/$(PACKAGE_NAME)/bpf/xdp/filter.c \
	                      -o /go/src/$(PACKAGE_NAME)/bpf/xdp/generated/xdp.ll && \
	               llc \
	                       -march=bpf \
	                       -filetype=obj \
	                       -o /go/src/$(PACKAGE_NAME)/bpf/xdp/generated/xdp.o \
	                       /go/src/$(PACKAGE_NAME)/bpf/xdp/generated/xdp.ll && \
	               rm -f /go/src/$(PACKAGE_NAME)/bpf/xdp/generated/xdp.ll"

.PHONY: packr
packr: bpf/bpf-packr.go bpf/packrd/packed-packr.go

bpf/bpf-packr.go bpf/packrd/packed-packr.go: bpf/xdp/generated/xdp.o
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
	          -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	              calico-build/bpf-clang \
	              /bin/sh -c \
	              "cd /go/src/$(PACKAGE_NAME)/bpf && /go/bin/packr2"

###############################################################################
# Building the image
###############################################################################
# Build the tigera/felix docker image, which contains only Felix.
.PHONY: $(BUILD_IMAGE) $(BUILD_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(BUILD_IMAGE)
$(BUILD_IMAGE): $(BUILD_IMAGE)-$(ARCH)
$(BUILD_IMAGE)-$(ARCH): bin/calico-felix-$(ARCH) register
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/calico-felix-$(ARCH) docker-image/bin/
	docker build --pull -t $(BUILD_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(BUILD_IMAGE):latest
endif

imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGES)))

sub-single-push-%:
	docker push $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

## push all arches
push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

## push multi-arch manifest where supported
push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
	# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*:$(IMAGETAG))-ARCH --target $(call unescapefs,$*:$(IMAGETAG))"

## push default amd64 arch where multi-arch manifest is not supported
push-non-manifests: imagetag $(addprefix sub-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))
sub-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker push $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of one arch for all supported registries
tag-images: imagetag $(addprefix sub-single-tag-images-arch-,$(call escapefs,$(PUSH_IMAGES))) $(addprefix sub-single-tag-images-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGES)))

sub-single-tag-images-arch-%:
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG)-$(ARCH))

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE):latest-$(ARCH) $(call unescapefs,$*:$(IMAGETAG))
else
	$(NOECHO) $(NOOP)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(VALIDARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

## tag version number build images i.e.  tigera/felix:latest-amd64 -> tigera/felix:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,quay.io/$(BUILD_IMAGE):$(VERSION)-$*)



###############################################################################
# Building OS Packages (debs/RPMS)
###############################################################################
# Build all the debs.
.PHONY: deb
deb: bin/calico-felix
ifeq ($(GIT_COMMIT),<unknown>)
	$(error Package builds must be done from a git working copy in order to calculate version numbers.)
endif
	$(MAKE) calico-build/trusty
	$(MAKE) calico-build/xenial
	$(MAKE) calico-build/bionic
	utils/make-packages.sh deb

# Build RPMs.
.PHONY: rpm
rpm: bin/calico-felix
ifeq ($(GIT_COMMIT),<unknown>)
	$(error Package builds must be done from a git working copy in order to calculate version numbers.)
endif
	$(MAKE) calico-build/centos7
ifneq ("$(ARCH)","ppc64le") # no ppc64le support in centos6
	$(MAKE) calico-build/centos6
endif
	utils/make-packages.sh rpm

# Build a docker image used for building debs for trusty.
.PHONY: calico-build/trusty
calico-build/trusty:
	cd docker-build-images && docker build -f ubuntu-trusty-build.Dockerfile.$(ARCH) -t calico-build/trusty .

# Build a docker image used for building debs for xenial.
.PHONY: calico-build/xenial
calico-build/xenial:
	cd docker-build-images && docker build -f ubuntu-xenial-build.Dockerfile.$(ARCH) -t calico-build/xenial .

# Build a docker image used for building debs for bionic.
.PHONY: calico-build/bionic
calico-build/bionic:
	cd docker-build-images && docker build -f ubuntu-bionic-build.Dockerfile.$(ARCH) -t calico-build/bionic .

# Construct a docker image for building Centos 7 RPMs.
.PHONY: calico-build/centos7
calico-build/centos7:
	cd docker-build-images && \
	  docker build \
	  --build-arg=UID=$(LOCAL_USER_ID) \
	  --build-arg=GID=$(LOCAL_GROUP_ID) \
	  -f centos7-build.Dockerfile.$(ARCH) \
	  -t calico-build/centos7 .

ifeq ("$(ARCH)","ppc64le")
	# Some commands that would typically be run at container build time must be run in a privileged container.
	@-docker rm -f centos7Tmp
	docker run --privileged --name=centos7Tmp calico-build/centos7 \
		/bin/bash -c "/setup-user; /install-centos-build-deps"
	docker commit centos7Tmp calico-build/centos7:latest
endif

# Construct a docker image for building Centos 6 RPMs.
.PHONY: calico-build/centos6
calico-build/centos6:
	cd docker-build-images && \
	  docker build \
	  --build-arg=UID=$(LOCAL_USER_ID) \
	  --build-arg=GID=$(LOCAL_GROUP_ID) \
	  -f centos6-build.Dockerfile.$(ARCH) \
	  -t calico-build/centos6 .




###############################################################################
# Managing the upstream library pins
###############################################################################

## Update dependency pins in glide.yaml
update-pins: update-licensing-pin update-typha-pin update-libcalico-pin

## deprecated target alias
update-libcalico: update-pins
	$(warning !! Update update-libcalico is deprecated, use update-pins !!)

update-typha: update-pins
	$(warning !! Update update-typha is deprecated, use update-pins !!)

## Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;


###############################################################################
## libcalico

## Set the default LIBCALICO source for this project
LIBCALICO_PROJECT_DEFAULT=tigera/libcalico-go-private.git
LIBCALICO_GLIDE_LABEL=projectcalico/libcalico-go

## Default the LIBCALICO repo and version but allow them to be overridden (master or release-vX.Y)
## default LIBCALICO branch to the same branch name as the current checked out repo
LIBCALICO_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
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
## typha

## Set the default TYPHA source for this project
TYPHA_PROJECT_DEFAULT=tigera/typha-private.git
TYPHA_GLIDE_LABEL=projectcalico/typha

## Default the TYPHA repo and version but allow them to be overridden (master or release-vX.Y)
## default TYPHA branch to the same branch name as the current checked out repo
TYPHA_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
TYPHA_REPO?=github.com/$(TYPHA_PROJECT_DEFAULT)
TYPHA_VERSION?=$(shell git ls-remote git@github.com:$(TYPHA_PROJECT_DEFAULT) $(TYPHA_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure TYPHA repo and branch are reachable
guard-git-typha:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(TYPHA_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(TYPHA_PROJECT_DEFAULT) "$(TYPHA_BRANCH)" "Ensure the branch exists, or set TYPHA_BRANCH variable";
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(TYPHA_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(TYPHA_PROJECT_DEFAULT) "$(TYPHA_BRANCH)" "Build container error, ensure ssh-agent is forwarding the correct keys."';
	@if [ "$(strip $(TYPHA_VERSION))" = "" ]; then \
		echo "ERROR: TYPHA version could not be determined"; \
		exit 1; \
	fi;

## Update libary pin in glide.yaml
update-typha-pin: guard-ssh-forwarding-bug guard-git-typha
	@$(DOCKER_RUN) $(TOOLING_BUILD) /bin/sh -c '\
		LABEL="$(TYPHA_GLIDE_LABEL)" \
		REPO="$(TYPHA_REPO)" \
		VERSION="$(TYPHA_VERSION)" \
		DEFAULT_REPO="$(TYPHA_PROJECT_DEFAULT)" \
		BRANCH="$(TYPHA_BRANCH)" \
		GLIDE="glide.yaml" \
		_scripts/update-pin.sh '

###############################################################################
## licensing

## Set the default LICENSING source for this project
LICENSING_PROJECT_DEFAULT=tigera/licensing
LICENSING_GLIDE_LABEL=tigera/licensing

## Default the LICENSING repo and version but allow them to be overridden (master or release-vX.Y)
## default LICENSING branch to the same branch name as the current checked out repo
LICENSING_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
LICENSING_REPO?=github.com/$(LICENSING_PROJECT_DEFAULT)
LICENSING_VERSION?=$(shell git ls-remote git@github.com:$(LICENSING_PROJECT_DEFAULT) $(LICENSING_BRANCH) 2>/dev/null | cut -f 1)

## Guard to ensure LICENSING repo and branch are reachable
guard-git-licensing:
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Ensure your ssh keys are correct and that you can access github" ;
	@_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "$(LICENSING_BRANCH)" "Ensure the branch exists, or set LICENSING_BRANCH variable";
	@$(DOCKER_RUN) $(TOOLING_BUILD) sh -c '_scripts/functions.sh ensure_can_reach_repo_branch $(LICENSING_PROJECT_DEFAULT) "master" "Build container error, ensure ssh-agent is forwarding the correct keys."';
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




###############################################################################
# Static checks
###############################################################################
.PHONY: static-checks
static-checks:
	$(MAKE) check-typha-pins go-meta-linter check-licenses check-packr

bin/check-licenses: $(SRC_FILES)
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) go build -v -i -o $@ "$(PACKAGE_NAME)/check-licenses"

.PHONY: check-licenses
check-licenses: check-licenses/dependency-licenses.txt bin/check-licenses
	@echo Checking dependency licenses
	$(DOCKER_RUN) $(CALICO_BUILD) bin/check-licenses

check-licenses/dependency-licenses.txt: vendor/.up-to-date
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'licenses ./cmd/calico-felix > check-licenses/dependency-licenses.txt'

.PHONY: go-meta-linter
go-meta-linter: vendor/.up-to-date $(GENERATED_FILES)
	# Run staticcheck stand-alone since gometalinter runs concurrent copies, which
	# uses a lot of RAM.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'glide nv | xargs -n 3 staticcheck'
	$(DOCKER_RUN) $(CALICO_BUILD) gometalinter --deadline=300s \
	                                --disable-all \
	                                --enable=goimports \
	                                --vendor ./...

.PHONY: check-packr
check-packr: bpf/packrd/packed-packr.go
	@if ! git diff --quiet bpf/packrd/packed-packr.go; then \
		echo "bpf/xdp/filter.c changed but the generated compiled object wasn't checked in. Please run 'make packr' and commit the changes to bpf/packrd/packed-packr.go."; \
		false; \
	fi

# Run go fmt on all our go files.
.PHONY: go-fmt goimports fix
fix go-fmt goimports:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'glide nv -x | \
	      grep -v -e "^\\.$$" | \
	      xargs goimports -w -local github.com/projectcalico/'

.PHONY: check-typha-pins
check-typha-pins: vendor/.up-to-date
	@echo "Checking Typha's libcalico-go pin matches ours (so that any datamodel"
	@echo "changes are reflected in the Typha-Felix API)."
	@echo
	@echo "Felix's libcalico-go pin:"
	@grep libcalico-go glide.lock -A 5 | grep 'version:' | head -n 1
	@echo "Typha's libcalico-go pin:"
	@grep libcalico-go vendor/github.com/projectcalico/typha/glide.lock -A 5 | grep 'version:' | head -n 1
	if [ "`grep libcalico-go glide.lock -A 5 | grep 'version:' | head -n 1`" != \
	     "`grep libcalico-go vendor/github.com/projectcalico/typha/glide.lock -A 5 | grep 'version:' | head -n 1`" ]; then \
	     echo "Typha and Felix libcalico-go pins differ."; \
	     false; \
	fi

.PHONY: pre-commit
pre-commit:
	$(DOCKER_RUN) $(CALICO_BUILD) git-hooks/pre-commit-in-container

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

foss-checks: vendor
	@echo Running $@...
	@docker run --rm -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -e FOSSA_API_KEY=$(FOSSA_API_KEY) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(CALICO_BUILD) /usr/local/bin/fossa

###############################################################################
# Unit Tests
###############################################################################
.PHONY: ut
ut combined.coverprofile: vendor/.up-to-date $(SRC_FILES)
	@echo Running Go UTs.
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) ./utils/run-coverage $(GINKGO_ARGS)

###############################################################################
# FV Tests
###############################################################################
fv/fv.test: vendor/.up-to-date $(SRC_FILES)
	# We pre-build the FV test binaries so that we can run them
	# outside a container and allow them to interact with docker.
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) go test ./$(shell dirname $@) -c --tags fvtests -o $@

.PHONY: fv
# runs all of the fv tests
# to run it in parallel, decide how many parallel engines you will run, and in each one call:
#         $(MAKE) fv FV_BATCHES_TO_RUN="<num>" FV_NUM_BATCHES=<num>
# where
#         FV_NUM_BATCHES = total parallel batches
#         FV_BATCHES_TO_RUN = which number this is
# e.g. to run it in 10 parallel runs:
#         $(MAKE) fv FV_BATCHES_TO_RUN="1" FV_NUM_BATCHES=10     # the first 1/10
#         $(MAKE) fv FV_BATCHES_TO_RUN="2" FV_NUM_BATCHES=10     # the second 1/10
#         $(MAKE) fv FV_BATCHES_TO_RUN="3" FV_NUM_BATCHES=10     # the third 1/10
#         ...
#         $(MAKE) fv FV_BATCHES_TO_RUN="10" FV_NUM_BATCHES=10    # the tenth 1/10
#         etc.
#
# To only run specific fv tests, set GINGKO_FOCUS to the desired Describe{}, Context{}
# or It{} description string. For example, to only run dns_test.go, type:
# 	GINKGO_FOCUS="DNS Policy" make fv
#
fv fv/latency.log: $(BUILD_IMAGE) bin/iptables-locker bin/test-workload bin/test-connection fv/fv.test
	docker build -t tigera-test/scapy fv/scapy
	cd fv && \
	  FV_FELIXIMAGE=$(FV_FELIXIMAGE) \
	  FV_ETCDIMAGE=$(FV_ETCDIMAGE) \
	  FV_TYPHAIMAGE=$(FV_TYPHAIMAGE) \
	  FV_K8SIMAGE=$(FV_K8SIMAGE) \
	  FV_NUM_BATCHES=$(FV_NUM_BATCHES) \
	  FV_BATCHES_TO_RUN="$(FV_BATCHES_TO_RUN)" \
	  PRIVATE_KEY=`pwd`/private.key \
	  GINKGO_ARGS='$(GINKGO_ARGS)' \
	  GINKGO_FOCUS="$(GINKGO_FOCUS)" \
	  ./run-batches
	@if [ -e fv/latency.log ]; then \
	   echo; \
	   echo "Latency results:"; \
	   echo; \
	   cat fv/latency.log; \
	fi

###############################################################################
# K8SFV Tests
###############################################################################
# Targets for Felix testing with the k8s backend and a k8s API server,
# with k8s model resources being injected by a separate test client.
GET_CONTAINER_IP := docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'
GRAFANA_VERSION=4.1.2
PROMETHEUS_DATA_DIR := $$HOME/prometheus-data
K8SFV_PROMETHEUS_DATA_DIR := $(PROMETHEUS_DATA_DIR)/k8sfv
$(K8SFV_PROMETHEUS_DATA_DIR):
	mkdir -p $@

# Directories that aren't part of the main Felix program,
# e.g. standalone test programs.
K8SFV_DIR:=k8sfv
NON_FELIX_DIRS:=$(K8SFV_DIR)
# Files for the Felix+k8s backend test program.
K8SFV_GO_FILES:=$(shell find ./$(K8SFV_DIR) -name prometheus -prune -o -type f -name '*.go' -print)

.PHONY: k8sfv-test k8sfv-test-existing-felix
# Run k8sfv test with Felix built from current code.
# control whether or not we use typha with USE_TYPHA=true or USE_TYPHA=false
# e.g.
#       $(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=true
#       $(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=false
k8sfv-test: $(BUILD_IMAGE) k8sfv-test-existing-felix
# Run k8sfv test with whatever is the existing 'tigera/felix:latest'
# container image.  To use some existing Felix version other than
# 'latest', do 'FELIX_VERSION=<...> make k8sfv-test-existing-felix'.
k8sfv-test-existing-felix: bin/k8sfv.test
	FV_ETCDIMAGE=$(FV_ETCDIMAGE) \
	FV_TYPHAIMAGE=$(FV_TYPHAIMAGE) \
	FV_FELIXIMAGE=$(FV_FELIXIMAGE) \
	FV_K8SIMAGE=$(FV_K8SIMAGE) \
	PRIVATE_KEY=`pwd`/fv/private.key \
	k8sfv/run-test

bin/k8sfv.test: $(K8SFV_GO_FILES) vendor/.up-to-date
	@echo Building $@...
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
	    sh -c 'go test -c -o $@ ./k8sfv && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: $@ was not statically linked"; false ) )'

.PHONY: run-prometheus run-grafana stop-prometheus stop-grafana
run-prometheus: stop-prometheus $(K8SFV_PROMETHEUS_DATA_DIR)
	FELIX_IP=`$(GET_CONTAINER_IP) k8sfv-felix` && \
	sed "s/__FELIX_IP__/$${FELIX_IP}/" < $(K8SFV_DIR)/prometheus/prometheus.yml.in > $(K8SFV_DIR)/prometheus/prometheus.yml
	docker run --detach --name k8sfv-prometheus \
	-v $(CURDIR)/$(K8SFV_DIR)/prometheus/prometheus.yml:/etc/prometheus.yml \
	-v $(K8SFV_PROMETHEUS_DATA_DIR):/prometheus \
	prom/prometheus \
	-config.file=/etc/prometheus.yml \
	-storage.local.path=/prometheus

stop-prometheus:
	@-docker rm -f k8sfv-prometheus
	sleep 2

run-grafana: stop-grafana run-prometheus
	docker run --detach --name k8sfv-grafana -p 3000:3000 \
	-v $(CURDIR)/$(K8SFV_DIR)/grafana:/etc/grafana \
	-v $(CURDIR)/$(K8SFV_DIR)/grafana-dashboards:/etc/grafana-dashboards \
	grafana/grafana:$(GRAFANA_VERSION) --config /etc/grafana/grafana.ini
	# Wait for it to get going.
	sleep 5
	# Configure prometheus data source.
	PROMETHEUS_IP=`$(GET_CONTAINER_IP) k8sfv-prometheus` && \
	sed "s/__PROMETHEUS_IP__/$${PROMETHEUS_IP}/" < $(K8SFV_DIR)/grafana-datasources/my-prom.json.in | \
	curl 'http://admin:admin@127.0.0.1:3000/api/datasources' -X POST \
	    -H 'Content-Type: application/json;charset=UTF-8' --data-binary @-

stop-grafana:
	@-docker rm -f k8sfv-grafana
	sleep 2

bin/iptables-locker: $(SRC_FILES) vendor/.up-to-date
	@echo Building iptables-locker...
	mkdir -p bin
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/fv/iptables-locker"'

bin/test-workload: $(SRC_FILES) vendor/.up-to-date
	@echo Building test-workload...
	mkdir -p bin
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/fv/test-workload"'

bin/test-connection: $(SRC_FILES) vendor/.up-to-date
	@echo Building test-connection...
	mkdir -p bin
	$(DOCKER_RUN) $(LOCAL_BUILD_MOUNTS) $(CALICO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/fv/test-connection"'

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
ci: image-all ut static-checks check-packr
ifeq (,$(filter fv, $(EXCEPT)))
	@$(MAKE) fv
endif
ifeq (,$(filter k8sfv-test, $(EXCEPT)))
	@$(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=true
	@$(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=false
endif

## Deploy images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(BRANCH_NAME) EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0)
GIT_VERSION?=$(shell git describe --tags --dirty)

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) release-windows-archive
	$(MAKE) VERSION=$(VERSION) tag-base-images-all
	$(MAKE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish"
	@echo ""
	@echo "Then, archive the Windows ZIP file, created at $(WINDOWS_ARCHIVE)."

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

## Produces the Windows ZIP archive for the release.
release-windows-archive $(WINDOWS_ARCHIVE): release-prereqs $(WINDOWS_ARCHIVE_FILES)
	-rm -f "$(WINDOWS_ARCHIVE)"
	mkdir -p dist
	zip --junk-paths "$(WINDOWS_ARCHIVE)" $(WINDOWS_ARCHIVE_FILES)

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	for img in $(BUILD_IMAGE):$(VERSION)-$(ARCH) quay.io/$(BUILD_IMAGE):$(VERSION)-$(ARCH); do \
	  if docker run $$img calico-felix --version | grep -q '$(VERSION)$$'; \
	  then \
	    echo "Check successful. ($$img)"; \
	  else \
	    echo "Incorrect version in docker image $$img!"; \
	    result=false; \
	  fi \
	done; \

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
	# Disabling for now since no-one is consuming the images.
	# $(MAKE) push-all IMAGETAG=$(VERSION)

	# Push binaries to GitHub release.
	# Requires ghr: https://github.com/tcnksm/ghr
	# Requires GITHUB_TOKEN environment variable set.
	ghr -u tigera -r felix-private \
		-b "Release notes can be found at https://docs.projectcalico.org" \
		-n $(VERSION) \
		$(VERSION) ./bin/

	@echo "Confirm that the release was published at the following URL."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "Build and publish the debs and rpms for this release."
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
	for img in $(BUILD_IMAGE):latest-$(ARCH) quay.io/$(BUILD_IMAGE):latest-$(ARCH); do \
	  if docker run $$img calico-felix --version | grep -q '$(VERSION)$$'; \
	  then \
	    echo "Check successful. ($$img)"; \
	  else \
	    echo "Incorrect version in docker image $$img!"; \
	    result=false; \
	  fi \
	done; \

	# Disabling for now since no-one is consuming the images.
	# $(MAKE) push-all IMAGETAG=latest

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
.PHONY: ut-no-cover
ut-no-cover: vendor/.up-to-date $(SRC_FILES)
	@echo Running Go UTs without coverage.
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo -r -skipPackage fv,k8sfv,windows $(GINKGO_ARGS)

.PHONY: ut-watch
ut-watch: vendor/.up-to-date $(SRC_FILES)
	@echo Watching go UTs for changes...
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo watch -r -skipPackage fv,k8sfv,windows $(GINKGO_ARGS)

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

bin/calico-felix.transfer-url: bin/calico-felix
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/calico-felix https://transfer.sh/calico-felix > $@'

.PHONY: patch-script
patch-script: bin/calico-felix.transfer-url
	$(DOCKER_RUN) $(CALICO_BUILD) bash -c 'utils/make-patch-script.sh $$(cat bin/calico-felix.transfer-url)'

# Generate a diagram of Felix's internal calculation graph.
docs/%.pdf: docs/%.dot
	cd docs/ && dot -Tpdf $*.dot -o $*.pdf

# Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/Masterminds/glide
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/gobuffalo/packr/v2/packr2

help:
	@echo "Felix Makefile"
	@echo
	@echo "Dependencies: docker 1.12+; go 1.7+"
	@echo
	@echo "Note: initial builds can be slow because they generate docker-based"
	@echo "build environments."
	@echo
	@echo "For any target, set ARCH=<target> to build for a given target."
	@echo "For example, to build for arm64:"
	@echo
	@echo "  make build ARCH=arm64"
	@echo
	@echo "To generate a docker image for arm64:"
	@echo
	@echo "  make image ARCH=arm64"
	@echo
	@echo "By default, builds for the architecture on which it is running. Cross-building is supported"
	@echo "only on amd64, i.e. building for other architectures when running on amd64."
	@echo "Supported target ARCH options:       $(ARCHES)"
	@echo
	@echo "Initial set-up:"
	@echo
	@echo "  make update-tools  Update/install the go build dependencies."
	@echo
	@echo "Builds:"
	@echo
	@echo "  make all                    Build all the binary packages."
	@echo "  make deb                    Build debs in ./dist."
	@echo "  make rpm                    Build rpms in ./dist."
	@echo "  make build                  Build binary."
	@echo "  make image                  Build docker image."
	@echo "  make build-all              Build binary for all supported architectures."
	@echo "  make image-all              Build docker images for all supported architectures."
	@echo "  make push IMAGETAG=tag      Deploy docker image with the tag IMAGETAG for the given ARCH, e.g. $(BUILD_IMAGE)<IMAGETAG>-<ARCH>."
	@echo "  make push-all IMAGETAG=tag  Deploy docker images with the tag IMAGETAG all supported architectures"
	@echo
	@echo "Tests:"
	@echo
	@echo "  make ut                Run UTs."
	@echo "  make fv                Run FVs."
	@echo "  make go-cover-browser  Display go code coverage in browser."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make update-vendor  Update the vendor directory with new "
	@echo "                      versions of upstream packages.  Record results"
	@echo "                      in glide.lock."
	@echo "  make go-fmt        Format our go code."
	@echo "  make clean         Remove binary files."
	@echo "-----------------------------------------"
	@echo "ARCH (target):          $(ARCH)"
	@echo "BUILDARCH (host):       $(BUILDARCH)"
	@echo "CALICO_BUILD:           $(CALICO_BUILD)"
	@echo "PROTOC_CONTAINER:       $(PROTOC_CONTAINER)"
	@echo "FV_ETCDIMAGE:           $(FV_ETCDIMAGE)"
	@echo "FV_K8SIMAGE:            $(FV_K8SIMAGE)"
	@echo "FV_TYPHAIMAGE:          $(FV_TYPHAIMAGE)"
	@echo "-----------------------------------------"
