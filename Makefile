# The build architecture is select by setting the ARCH variable.
# For example: When building on ppc64le you could use ARCH=ppc64le make <....>.
# When ARCH is undefined it defaults to amd64.
ifdef ARCH
	ARCHTAG:=-$(ARCH)
endif
ARCH?=amd64
ARCHTAG?=

ifeq ($(ARCH),amd64)
GO_BUILD_VER:=v0.6
endif

ifeq ($(ARCH),ppc64le)
GO_BUILD_VER:=latest
endif

GO_BUILD_CONTAINER?=calico/go-build$(ARCHTAG):$(GO_BUILD_VER)

help:
	@echo "Calico K8sapiserver Makefile"
	@echo "Builds:"
	@echo
	@echo "  make all                  Build all the binary packages."
	@echo "  make calico/k8sapiserver  Build calico/k8sapiserver docker image."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make test                Run Tests."
	@echo "Maintenance:"
	@echo
	@echo "  make clean         Remove binary files."
# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

all: calico/k8sapiserver
test: ut fv

# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
GIT_COMMIT:=$(shell git rev-parse HEAD || echo '<unknown>')
BUILD_ID:=$(shell git rev-parse HEAD || uuidgen | sed 's/-//g')
GIT_DESCRIPTION:=$(shell git describe --tags || echo '<unknown>')

# Define some constants
#######################
BINDIR        ?= bin
BUILD_DIR     ?= build
CAPI_PKG       = github.com/tigera/calico-k8sapiserver
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
K8SAPISERVER_GO_FILE = $(shell find $(SRC_DIRS) -name \*.go -exec $(STAT) {} \; \
                   | sort -r | head -n 1 | sed "s/.* //")
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -v $${PWD}:/go/src/github.com/tigera/calico-k8sapiserver:rw \
                              -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
                              -w /go/src/github.com/tigera/calico-k8sapiserver \
                              $(GO_BUILD_CONTAINER)

# Linker flags for building Felix.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
        -X github.com/tigera/calico-k8sapiserver/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
        -X github.com/tigera/calico-k8sapiserver/buildinfo.BuildDate=$(DATE) \
        -X github.com/tigera/calico-k8sapiserver/buildinfo.GitRevision=$(GIT_COMMIT) \
        -B 0x$(BUILD_ID)"

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

$(BINDIR)/defaulter-gen: 
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/defaulter-gen'

$(BINDIR)/deepcopy-gen:
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/deepcopy-gen'

$(BINDIR)/conversion-gen: 
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/conversion-gen'

$(BINDIR)/client-gen:
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/client-gen'

$(BINDIR)/lister-gen:
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/lister-gen'

$(BINDIR)/informer-gen:
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/informer-gen'

$(BINDIR)/openapi-gen: vendor/k8s.io/kubernetes/cmd/libs/go2idl/openapi-gen
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -o $@ $(CAPI_PKG)/$^'

# Regenerate all files if the gen exes changed or any "types.go" files changed
.generate_files: .generate_exes $(TYPES_FILES)
	# Generate defaults
	$(BINDIR)/defaulter-gen \
		--v 1 --logtostderr \
		--go-header-file "vendor/github.com/kubernetes/repo-infra/verify/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico/v2" \
	  	--extra-peer-dirs "$(CAPI_PKG)/pkg/apis/calico" \
		--extra-peer-dirs "$(CAPI_PKG)/pkg/apis/calico/v2" \
		--output-file-base "zz_generated.defaults"
	# Generate deep copies
	$(BINDIR)/deepcopy-gen \
		--v 1 --logtostderr \
		--go-header-file "vendor/github.com/kubernetes/repo-infra/verify/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico/v2" \
		--bounding-dirs "github.com/tigera/calico-k8sapiserver" \
		--output-file-base zz_generated.deepcopy
	# Generate conversions
	$(BINDIR)/conversion-gen \
		--v 1 --logtostderr \
		--go-header-file "vendor/github.com/kubernetes/repo-infra/verify/boilerplate/boilerplate.go.txt" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico" \
		--input-dirs "$(CAPI_PKG)/pkg/apis/calico/v2" \
		--output-file-base zz_generated.conversion
	# generate all pkg/client contents
	$(DOCKER_CMD) $(BUILD_DIR)/update-client-gen.sh
	touch $@

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "apiserver" instead of "$(BINDIR)/apiserver".
#########################################################################
$(BINDIR)/calico-k8sapiserver: .generate_files $(K8SAPISERVER_GO_FILES)
	@echo Building k8sapiserver...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "$(CAPI_PKG)/cmd/apiserver" && \
               ( ldd $(BINDIR)/calico-k8sapiserver 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: $(BINDIR)/calico-k8sapiserver was not statically linked"; false ) )'

# Build the calico/k8sapiserver docker image.
.PHONY: calico/k8sapiserver
calico/k8sapiserver: .generate_files \
    $(BINDIR)/calico-k8sapiserver
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp $(BINDIR)/calico-k8sapiserver docker-image/bin/
	docker build --pull -t calico/k8sapiserver docker-image

.PHONY: ut
ut: run-etcd
	$(DOCKER_GO_BUILD) \
		sh -c 'go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(CAPI_PKG)/,$(TEST_DIRS))'

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

.PHONY: clean
clean: clean-bin clean-build-image clean-generated
clean-build-image:
	docker rmi -f calico-k8sapiserver > /dev/null 2>&1 || true

clean-generated:
	rm -f .generate_files
	find $(TOP_SRC_DIRS) -name zz_generated* -exec rm {} \;
	# rollback changes to the generated clientset directories
	# find $(TOP_SRC_DIRS) -type d -name *_generated -exec rm -rf {} \;

clean-bin:
	rm -rf $(BINDIR) \
		   .generate_exes \
	       docker-image/bin