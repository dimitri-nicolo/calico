# This Makefile builds Felix and releases it in various forms:
#
#					 		     Go install
#							         |
#					    +-------+	         v
#					    | Felix |   +-----------------+
#					    |  Go   |   | calico/go-build |
#					    |  code |   +-----------------+
#					    +-------+	    /
#						    \      /
#						     \    /
#						     go build
# +----------------------+				|
# | calico-build/centos7 |				v
# | calico-build/xenial  |			 +------------------+
# | calico-build/trusty  |			 | bin/calico-felix |
# +----------------------+			 +------------------+
#		     \				  /	|
#		      \	     	   --------------/	|
#		       \	  /			|
#			\	 /			|
#			 \      /			|
#		     rpm/build-rpms			|
#		   debian/build-debs			|
#			   |				|
#			   |			  docker build
#			   v				|
#	    +----------------------------+		|
#	    |  RPM packages for Centos7  |		|
#	    |  RPM packages for Centos6  |		v
#	    | Debian packages for Xenial |      +--------------+
#	    | Debian packages for Trusty |      | tigera/felix |
#	    +----------------------------+      +--------------+
#
###############################################################################
PACKAGE_NAME?=github.com/projectcalico/felix
GO_BUILD_VER?=v0.53

GIT_USE_SSH = true
LOCAL_CHECKS = check-typha-pins

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_FELIX_PRIVATE_PROJECT_ID)

SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_NODE_PRIVATE_PROJECT_ID) $(SEMAPHORE_KUBE_CONTROLLERS_PRIVATE_PROJECT_ID) \
	$(SEMAPHORE_TS_QUERYSERVER_PROJECT_ID) $(SEMAPHORE_CALICOQ_PROJECT_ID) $(SEMAPHORE_CLOUD_CONTROLLERS_PRIVATE_PROJECT_ID)

# Build mounts for running in "local build" mode. This allows an easy build using local development code,
# assuming that there is a local checkout of libcalico in the same directory as this repo.
ifdef LOCAL_BUILD
PHONY: set-up-local-build
LOCAL_BUILD_DEP:=set-up-local-build

EXTRA_DOCKER_ARGS+=-v $(CURDIR)/../libcalico-go-private:/go/src/github.com/projectcalico/libcalico-go-private:rw \
	-v $(CURDIR)/../typha-private:/go/src/github.com/projectcalico/typha-private:rw \
	-v $(CURDIR)/../pod2daemon:/go/src/github.com/projectcalico/pod2daemon:rw

$(LOCAL_BUILD_DEP):
	$(DOCKER_GO_BUILD) go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go-private \
	-replace=github.com/projectcalico/typha=../typha-private \
	-replace=github.com/projectcalico/pod2daemon=../pod2daemon
endif

FELIX_IMAGE        ?=tigera/felix
BUILD_IMAGES       ?=$(FELIX_IMAGE)
DEV_REGISTRIES     ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES ?=quay.io
RELEASE_BRANCH_PREFIX ?= release-calient
DEV_TAG_SUFFIX        ?= calient-0.dev

# All Felix go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_FELIX_DIRS) fv,-path ./$(dir) -prune -o) -type f -name '*.go' -print) $(GENERATED_FILES)
FV_SRC_FILES:=$(shell find fv -type f -name '*.go' -print)
EXTRA_DOCKER_ARGS += --init -v $(CURDIR)/../pod2daemon:/go/src/github.com/projectcalico/pod2daemon:rw
EXTRA_DOCKER_ARGS += --init -e GOPRIVATE=github.com/tigera/*

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

include Makefile.common

FV_ETCDIMAGE  ?=quay.io/coreos/etcd:$(ETCD_VERSION)-$(BUILDARCH)
FV_K8SIMAGE   ?=gcr.io/google_containers/hyperkube-$(BUILDARCH):$(K8S_VERSION)
FV_TYPHAIMAGE ?=calico/typha:master-$(BUILDARCH)
FV_FELIXIMAGE ?=$(FELIX_IMAGE)-test:latest-$(BUILDARCH)

# If building on amd64 omit the arch in the container name.  Fixme!
ifeq ($(BUILDARCH),amd64)
	FV_ETCDIMAGE=quay.io/coreos/etcd:$(ETCD_VERSION)
	FV_K8SIMAGE=gcr.io/google_containers/hyperkube:$(K8S_VERSION)
	FV_TYPHAIMAGE=gcr.io/unique-caldron-775/cnx/tigera/typha:master
endif

# Total number of batches to split the tests into.  In CI we set this to say 5 batches,
# and run a single batch on each test VM.
FV_NUM_BATCHES?=1
# Space-delimited list of FV batches to run in parallel.  Defaults to running all batches
# in parallel on this host.  The CI system runs a subset of batches in each parallel job.
#
# To run multiple batches in parallel in order to speed up local runs (on a powerful
# developer machine), set FV_NUM_BATCHES to say 3, then this value will be automatically
# calculated.  Note: the tests tend to flake more when run in parallel even though they
# were designed to be isolated; if you hit a failure, try running the tests sequentially
# (with FV_NUM_BATCHES=1) to check that it's not a flake.
FV_BATCHES_TO_RUN?=$(shell seq $(FV_NUM_BATCHES))
FV_SLOW_SPEC_THRESH=90
FV_RACE_DETECTOR_ENABLED?=false

# Linker flags for building Felix.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS=-ldflags "\
	-X $(PACKAGE_NAME)/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/buildinfo.BuildDate=$(DATE) \
	-X $(PACKAGE_NAME)/buildinfo.GitRevision=$(GIT_COMMIT) \
	-B 0x$(BUILD_ID)"

# List of Go files that are generated by the build process.  Builds should
# depend on these, clean removes them.
GENERATED_FILES:=proto/felixbackend.pb.go bpf/asm/opcode_string.go

# Files to include in the Windows ZIP archive.
WINDOWS_ARCHIVE_FILES := bin/tigera-felix.exe windows-packaging/README.txt windows-packaging/*.ps1
# Name of the Windows release ZIP archive.
WINDOWS_ARCHIVE := dist/tigera-felix-windows-$(VERSION).zip

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/bin \
	       dist \
	       build \
	       fv/fv.test \
	       $(GENERATED_FILES) \
	       go/docs/calc.pdf \
	       release-notes-* \
	       fv/infrastructure/crds/ \
	       docs/*.pdf \
	       .go-pkg-cache \
	       vendor \
	       Makefile.common*
	find . -name "junit.xml" -type f -delete
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete
	$(DOCKER_GO_BUILD) make -C bpf-apache clean
	$(DOCKER_GO_BUILD) make -C bpf-gpl clean
	-docker rmi $(FELIX_IMAGE)-wgtool:latest-amd64
	-docker rmi $(FELIX_IMAGE)-wgtool:latest

###############################################################################
# Updating pins
###############################################################################
LICENSING_BRANCH?=$(PIN_BRANCH)
LICENSING_REPO?=github.com/tigera/licensing
LIBCALICO_REPO=github.com/tigera/libcalico-go-private
TYPHA_REPO=github.com/tigera/typha-private

update-licensing-pin:
	$(call update_pin,github.com/tigera/licensing,$(LICENSING_REPO),$(LICENSING_BRANCH))

update-pins: update-licensing-pin update-pod2daemon-pin replace-libcalico-pin replace-typha-pin

POD2DAEMON_BRANCH?=$(PIN_BRANCH)
POD2DAEMON_REPO?=github.com/projectcalico/pod2daemon

update-pod2daemon-pin:
	$(call update_pin,github.com/projectcalico/pod2daemon,$(POD2DAEMON_REPO),$(POD2DAEMON_BRANCH))

NFNETLINK_BRANCH?=$(PIN_BRANCH)
NFNETLINK_REPO?=github.com/tigera/nfnetlink

update-nfnetlink-pin:
	$(call update_pin,github.com/tigera/nfnetlink,$(NFNETLINK_REPO),$(NFNETLINK_BRANCH))

###############################################################################
# Building the binary
###############################################################################
build: bin/calico-felix build-bpf bin/calico-felix.exe
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/calico-felix: bin/calico-felix-$(ARCH)
	ln -f bin/calico-felix-$(ARCH) bin/calico-felix

ifeq ($(ARCH), amd64)
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

DOCKER_GO_BUILD_CGO=$(DOCKER_RUN) -e CGO_ENABLED=$(CGO_ENABLED) $(CALICO_BUILD)

bin/calico-felix-$(ARCH): $(SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building felix for $(ARCH) on $(BUILDARCH)
	mkdir -p bin
	if [ "$(SEMAPHORE)" != "true" -o ! -e $@ ] ; then \
	  $(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
	     go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-felix"'; \
	fi

# Cross-compile Felix for Windows
bin/calico-felix.exe: $(SRC_FILES)
	@echo Building felix for Windows...
	mkdir -p bin
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
	   	GOOS=windows CC=x86_64-w64-mingw32-gcc go build --buildmode=exe -v -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-felix" && \
		( ldd $@ 2>&1 | grep -q "Not a valid dynamic program\|not a dynamic executable" || \
		( echo "Error: $@ was not statically linked"; false ) )'

bin/tigera-felix.exe: $(REMOTE_DEPS) bin/calico-felix.exe
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

bin/calico-felix-race-$(ARCH): $(SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building felix with race detector enabled for $(ARCH) on $(BUILDARCH)
	mkdir -p bin
	if [ "$(SEMAPHORE)" != "true" -o ! -e $@ ] ; then \
	  $(DOCKER_GO_BUILD_CGO) \
	     sh -c 'go build -v -race -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-felix"'; \
	fi

# Generate the protobuf bindings for go. The proto/felixbackend.pb.go file is included in SRC_FILES
protobuf proto/felixbackend.pb.go: proto/felixbackend.proto
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		  -v $(CURDIR):/code -v $(CURDIR)/proto:/src:rw \
		      $(PROTOC_CONTAINER) \
		      --gogofaster_out=plugins=grpc:. \
		      felixbackend.proto
	# Make sure the generated code won't cause a static-checks failure.
	$(MAKE) fix

# We pre-build lots of different variants of the TC programs, defer to the script.
BPF_GPL_O_FILES:=$(addprefix bpf-gpl/,$(shell bpf-gpl/list-objs))

# There's a one-to-one mapping from UT C files to objects and the same for the apache programs..
BPF_GPL_UT_O_FILES:=$(BPF_GPL_UT_C_FILES:.c=.o) $(addprefix bpf-gpl/,$(shell bpf-gpl/list-ut-objs))
BPF_APACHE_C_FILES:=$(wildcard bpf-apache/*.c)
BPF_APACHE_O_FILES:=$(addprefix bpf-apache/bin/,$(notdir $(BPF_APACHE_C_FILES:.c=.o)))

ALL_BPF_PROGS=$(BPF_GPL_O_FILES) $(BPF_APACHE_O_FILES)

# Mark the BPF programs phony so we'll always defer to their own makefile.  This is OK as long as
# we're only depending on the BPF programs from other phony targets.  (Otherwise, we'd do
# unnecessary rebuilds of anything that depends on the BPF prgrams.)
.PHONY: build-bpf clean-bpf
build-bpf:
	$(DOCKER_GO_BUILD) sh -c "make -j -C bpf-apache all && \
	                          make -j -C bpf-gpl all ut-objs ARCH=$(ARCH)"

clean-bpf:
	rm -f bpf-gpl/*.d bpf-apache/*.d
	$(DOCKER_GO_BUILD) sh -c "make -j -C bpf-apache clean && \
	                          make -j -C bpf-gpl clean"

bpf/asm/opcode_string.go: bpf/asm/asm.go
	$(DOCKER_GO_BUILD) go generate ./bpf/asm/

###############################################################################
# Building the image
###############################################################################
# Build the calico/felix docker image, which contains only Felix.
.PHONY: $(FELIX_IMAGE) $(FELIX_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(FELIX_IMAGE)
$(FELIX_IMAGE): $(FELIX_IMAGE)-$(ARCH)
$(FELIX_IMAGE)-$(ARCH): bin/calico-felix-$(ARCH) \
                        bin/calico-bpf \
                        build-bpf \
                        docker-image/calico-felix-wrapper \
                        docker-image/felix.cfg \
                        docker-image/Dockerfile* \
                        register
	# Reconstruct the bin and bpf directories because we don't want to accidentally add
	# leftover files (say from a build on another branch) into the docker image.
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/calico-felix-$(ARCH) docker-image/bin/
	cp bin/calico-bpf docker-image/bin/
	rm -rf docker-image/bpf
	mkdir -p docker-image/bpf/bin
	# Copy only the files we're explicitly expecting (in case we have left overs after switching branch).
	cp $(ALL_BPF_PROGS) docker-image/bpf/bin
	docker build --pull -t $(FELIX_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/Dockerfile.$(ARCH) docker-image;
ifeq ($(ARCH),amd64)
	docker tag $(FELIX_IMAGE):latest-$(ARCH) $(FELIX_IMAGE):latest
endif

ifeq ($(FV_RACE_DETECTOR_ENABLED),true)
FV_BINARY=calico-felix-race-amd64
else
FV_BINARY=calico-felix-amd64
endif

image-test: image fv/Dockerfile.test.amd64 bin/pktgen bin/test-workload bin/test-connection bin/tproxy bin/$(FV_BINARY) image-wgtool
	docker build -t $(FELIX_IMAGE)-test:latest-$(ARCH) --build-arg FV_BINARY=$(FV_BINARY) --file ./fv/Dockerfile.test.$(ARCH) bin;
ifeq ($(ARCH),amd64)
	docker tag $(FELIX_IMAGE)-test:latest-$(ARCH) $(FELIX_IMAGE)-test:latest
endif

image-wgtool: fv/Dockerfile.wgtool.amd64
	docker build -t $(FELIX_IMAGE)-wgtool:latest-$(ARCH) --file ./fv/Dockerfile.wgtool.$(ARCH) fv;
ifeq ($(ARCH),amd64)
	docker tag $(FELIX_IMAGE)-wgtool:latest-$(ARCH) $(FELIX_IMAGE)-wgtool:latest
endif


## tag version number build images i.e.  tigera/felix:latest-amd64 -> tigera/felix:v1.1.1-amd64
tag-base-images-all: $(addprefix sub-base-tag-images-,$(VALIDARCHES))
sub-base-tag-images-%:
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,$(BUILD_IMAGE):$(VERSION)-$*)
	docker tag $(BUILD_IMAGE):latest-$* $(call unescapefs,quay.io/$(BUILD_IMAGE):$(VERSION)-$*)

###############################################################################
# Static checks
###############################################################################
# FIXME re-enable linting once we figure out why the linter barfs on this repo.
LINT_ARGS = --disable gosimple,unused,staticcheck,govet,errcheck,structcheck,varcheck,deadcode,ineffassign

LOCAL_CHECKS = check-typha-pins

LIBCALICO_FELIX?=$(shell $(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download; go list -m -f "{{.Replace.Version}}" github.com/projectcalico/libcalico-go')
TYPHA_GOMOD_DIR?=$(shell $(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download; go list -m -f "{{.Dir}}" github.com/projectcalico/typha')
ifneq ($(TYPHA_GOMOD_DIR),)
	LIBCALICO_TYPHA?=$(shell $(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download; (cd $(TYPHA_GOMOD_DIR); go list -m -f "{{.Replace.Version}}" github.com/projectcalico/libcalico-go)')
endif

.PHONY: check-typha-pins
check-typha-pins:
	@echo "Checking Typha's libcalico-go pin matches ours (so that any datamodel"
	@echo "changes are reflected in the Typha-Felix API)."
	@echo
	@echo "Felix's libcalico-go pin: $(LIBCALICO_FELIX)"
	@echo "Typha's libcalico-go pin: $(LIBCALICO_TYPHA)"
	if [ "$(LIBCALICO_FELIX)" != "$(LIBCALICO_TYPHA)" ]; then \
	     echo "Typha and Felix libcalico-go pins differ."; \
	     false; \
	fi

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: golangci-lint
golangci-lint: $(GENERATED_FILES)
	$(DOCKER_GO_BUILD_CGO) golangci-lint run $(LINT_ARGS)

###############################################################################
# Unit Tests
###############################################################################

UT_PACKAGES_TO_SKIP?=fv,k8sfv,bpf/ut

.PHONY: ut
ut combined.coverprofile: $(SRC_FILES) build-bpf
	@echo Running Go UTs.
	$(DOCKER_GO_BUILD_CGO) ./utils/run-coverage -skipPackage $(UT_PACKAGES_TO_SKIP) $(GINKGO_ARGS)

###############################################################################
# FV Tests
###############################################################################
fv/fv.test: $(SRC_FILES) $(FV_SRC_FILES)
	# We pre-build the FV test binaries so that we can run them
	# outside a container and allow them to interact with docker.
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) go test $(BUILD_FLAGS) ./$(shell dirname $@) -c --tags fvtests -o $@'

REMOTE_DEPS=fv/infrastructure/crds

fv/infrastructure/crds: go.mod go.sum $(LOCAL_BUILD_DEP)
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	go list all; \
	cp -r `go list -m -f "{{.Dir}}" github.com/projectcalico/libcalico-go`/config/crd fv/infrastructure/crds; \
	chmod +w fv/infrastructure/crds/'

.PHONY: fv
# runs all of the fv tests
# to run it in parallel, decide how many parallel engines you will run, and in each one call:
#	 $(MAKE) fv FV_BATCHES_TO_RUN="<num>" FV_NUM_BATCHES=<num>
# where
#	 FV_NUM_BATCHES = total parallel batches
#	 FV_BATCHES_TO_RUN = which number this is
# e.g. to run it in 10 parallel runs:
#	 $(MAKE) fv FV_BATCHES_TO_RUN="1" FV_NUM_BATCHES=10     # the first 1/10
#	 $(MAKE) fv FV_BATCHES_TO_RUN="2" FV_NUM_BATCHES=10     # the second 1/10
#	 $(MAKE) fv FV_BATCHES_TO_RUN="3" FV_NUM_BATCHES=10     # the third 1/10
#	 ...
#	 $(MAKE) fv FV_BATCHES_TO_RUN="10" FV_NUM_BATCHES=10    # the tenth 1/10
#	 etc.
#
# To only run specific fv tests, set GINGKO_FOCUS to the desired Describe{}, Context{}
# or It{} description string. For example, to only run dns_test.go, type:
# 	GINKGO_FOCUS="DNS Policy" make fv
#
fv fv/latency.log fv/data-races.log: $(REMOTE_DEPS) image-test bin/iptables-locker bin/test-workload bin/test-connection bin/calico-bpf fv/fv.test
	rm -f fv/data-races.log fv/latency.log
	docker build -t tigera-test/scapy fv/scapy
	cd fv && \
	  FV_FELIXIMAGE=$(FV_FELIXIMAGE) \
	  FV_ETCDIMAGE=$(FV_ETCDIMAGE) \
	  FV_TYPHAIMAGE=$(FV_TYPHAIMAGE) \
	  FV_K8SIMAGE=$(FV_K8SIMAGE) \
	  FV_NUM_BATCHES=$(FV_NUM_BATCHES) \
	  FV_BATCHES_TO_RUN="$(FV_BATCHES_TO_RUN)" \
	  FELIX_FV_FELIX_LOG_LEVEL="$(FELIX_FV_FELIX_LOG_LEVEL)" \
	  PRIVATE_KEY=`pwd`/private.key \
	  GINKGO_ARGS='$(GINKGO_ARGS)' \
	  GINKGO_FOCUS="$(GINKGO_FOCUS)" \
	  FELIX_FV_ENABLE_BPF="$(FELIX_FV_ENABLE_BPF)" \
	  FV_RACE_DETECTOR_ENABLED=$(FV_RACE_DETECTOR_ENABLED) \
	  FELIX_FV_WIREGUARD_AVAILABLE=`./wireguard-available >/dev/null && echo true || echo false` \
	  ./run-batches
	@if [ -e fv/latency.log ]; then \
	   echo; \
	   echo "Latency results:"; \
	   echo; \
	   cat fv/latency.log; \
	fi

fv-bpf:
	$(MAKE) fv FELIX_FV_ENABLE_BPF=true

check-wireguard:
	fv/wireguard-available || ( echo "WireGuard not available."; exit 1 )

fv/win-fv.exe: $(REMOTE_DEPS)
	mkdir -p bin
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
	   	GOOS=windows CC=x86_64-w64-mingw32-gcc go test --buildmode=exe -mod=mod ./$(shell dirname $@)/winfv -c -o $@'

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
k8sfv-test: image-test k8sfv-test-existing-felix
# Run k8sfv test with whatever is the existing 'tigera/felix:latest'
# container image.  To use some existing Felix version other than
# 'latest', do 'FELIX_VERSION=<...> make k8sfv-test-existing-felix'.
k8sfv-test-existing-felix: $(REMOTE_DEPS) bin/k8sfv.test
	FV_ETCDIMAGE=$(FV_ETCDIMAGE) \
	FV_TYPHAIMAGE=$(FV_TYPHAIMAGE) \
	FV_FELIXIMAGE=$(FV_FELIXIMAGE) \
	FV_K8SIMAGE=$(FV_K8SIMAGE) \
	PRIVATE_KEY=`pwd`/fv/private.key \
	k8sfv/run-test

bin/k8sfv.test: $(K8SFV_GO_FILES)
	@echo Building $@...
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		go test -c $(BUILD_FLAGS) -o $@ ./k8sfv'

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

bin/calico-bpf: $(SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building calico-bpf...
	$(DOCKER_GO_BUILD_CGO) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/calico-bpf"'

bin/pktgen: $(SRC_FILES) $(FV_SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building pktgen...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/fv/pktgen"'

bin/tproxy: $(SRC_FILES) $(FV_SRC_FILES) $(LOCAL_BUILD_DEP)
	@echo Building tproxy...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/fv/tproxy/cmd"'

bin/iptables-locker: $(LOCAL_BUILD_DEP) go.mod $(shell find iptables -type f -name '*.go' -print)
	@echo Building iptables-locker...
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/fv/iptables-locker"'

bin/test-workload: $(LOCAL_BUILD_DEP) go.mod fv/cgroup/cgroup.go fv/utils/utils.go fv/connectivity/*.go fv/test-workload/*.go
	@echo Building test-workload...
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/fv/test-workload"'

bin/test-connection: $(LOCAL_BUILD_DEP) go.mod fv/cgroup/cgroup.go fv/utils/utils.go fv/connectivity/*.go fv/test-connection/*.go
	@echo Building test-connection...
	mkdir -p bin
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/fv/test-connection"'

st:
	@echo "No STs available"

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

ci: mod-download image-all ut
ifeq (,$(filter fv, $(EXCEPT)))
	@$(MAKE) fv
endif
ifeq (,$(filter k8sfv-test, $(EXCEPT)))
	@$(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=true
	@$(MAKE) k8sfv-test JUST_A_MINUTE=true USE_TYPHA=false
endif

## Deploy images to registry
cd: cd-common

## Vendor is now a no-op, but kept in place for backwards compatibility in our semaphore jobs.
.PHONY: vendor
vendor:
	@echo "vendoring not required for gomod"

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: ut-no-cover
ut-no-cover: $(SRC_FILES)
	@echo Running Go UTs without coverage.
	$(DOCKER_GO_BUILD) ginkgo -r -skipPackage $(UT_PACKAGES_TO_SKIP) $(GINKGO_ARGS)

.PHONY: ut-watch
ut-watch: $(SRC_FILES)
	@echo Watching go UTs for changes...
	$(DOCKER_GO_BUILD) ginkgo watch -r -skipPackage $(UT_PACKAGES_TO_SKIP) $(GINKGO_ARGS)

.PHONY: bin/bpf.test
bin/bpf.test: $(GENERATED_FILES) $(shell find bpf/ -name '*.go')
	$(DOCKER_GO_BUILD_CGO) go test $(BUILD_FLAGS) ./bpf/ -c -o $@

.PHONY: bin/bpf_ut.test
bin/bpf_ut.test: $(GENERATED_FILES) $(shell find bpf/ -name '*.go')
	$(DOCKER_GO_BUILD_CGO) go test $(BUILD_FLAGS) ./bpf/ut -c -o $@

# Build debug version of bpf.test for use with the delve debugger.
.PHONY: bin/bpf_debug.test
bin/bpf_debug.test: $(GENERATED_FILES) $(shell find bpf/ -name '*.go')
	$(DOCKER_GO_BUILD_CGO) go test $(BUILD_FLAGS) ./bpf/ut -c -gcflags="-N -l" -o $@

.PHONY: ut-bpf
ut-bpf: bin/bpf_ut.test bin/bpf.test build-bpf
	$(DOCKER_RUN) \
		--privileged \
		-e RUN_AS_ROOT=true \
		$(CALICO_BUILD) sh -c ' \
		mount bpffs /sys/fs/bpf -t bpf && \
		cd /go/src/$(PACKAGE_NAME)/bpf/ && \
		BPF_FORCE_REAL_LIB=true ../bin/bpf.test -test.v -test.run "$(FOCUS)"'
	$(DOCKER_RUN) \
		--privileged \
		-e RUN_AS_ROOT=true \
		-v `pwd`:/code \
		-v `pwd`/bpf-gpl/bin:/usr/lib/calico/bpf \
		$(CALICO_BUILD) sh -c ' \
		mount bpffs /sys/fs/bpf -t bpf && \
		cd /go/src/$(PACKAGE_NAME)/bpf/ut && \
		../../bin/bpf_ut.test -test.v -test.run "$(FOCUS)"'

## Launch a browser with Go coverage stats for the whole project.
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
	@$(DOCKER_GO_BUILD) sh -c 'go tool cover -func combined.coverprofile | \
				   sed 's=$(PACKAGE_NAME)/==' | \
				   column -t'
	@echo
	@echo ======== Missing coverage only =========
	@echo
	@$(DOCKER_GO_BUILD) sh -c "go tool cover -func combined.coverprofile | \
				   sed 's=$(PACKAGE_NAME)/==' | \
				   column -t | \
				   grep -v '100\.0%'"

bin/calico-felix.transfer-url: bin/calico-felix
	$(DOCKER_GO_BUILD) sh -c 'curl --upload-file bin/calico-felix https://transfer.sh/calico-felix > $@'


.PHONY: patch-script
patch-script: bin/calico-felix.transfer-url
	$(DOCKER_GO_BUILD) bash -c 'utils/make-patch-script.sh $$(cat bin/calico-felix.transfer-url)'

# Generate diagrams showing Felix internals:
# - docs/calc.pdf: Felix's internal calculation graph.
# - docs/flowlogs.pdf: Structures involved in flow log processing.
# - docs/dnslogs.pdf: Structures involved in DNS log processing.
docs/%.pdf: docs/%.dot
	cd docs/ && dot -Tpdf $*.dot -o $*.pdf

## Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/onsi/ginkgo/ginkgo
