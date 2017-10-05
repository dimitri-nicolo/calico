# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all: build

# Some env vars that devs might find useful:
#  GOFLAGS      : extra "go build" flags to use - e.g. -v   (for verbose)
#  NO_DOCKER=1  : execute each step natively, not in a Docker container
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Define some constants
#######################
ROOT           = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BINDIR        ?= bin
BUILD_DIR     ?= build
ARTIFACTS     ?= artifacts
CAPI_PKG       = github.com/tigera/calico-k8sapiserver
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
VERSION       ?= $(shell git describe --always --abbrev=7 --dirty)
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
NEWEST_GO_FILE = $(shell find $(SRC_DIRS) -name \*.go -exec $(STAT) {} \; \
                   | sort -r | head -n 1 | sed "s/.* //")
TYPES_FILES    = $(shell find pkg/apis -name types.go)
GO_VERSION     = 1.7.3

PLATFORM?=linux
ARCH?=amd64

GO_BUILD       = env GOOS=$(PLATFORM) GOARCH=$(ARCH) go build -i $(GOFLAGS) \
                   -ldflags "-X $(CAPI_PKG)/pkg.VERSION=$(VERSION)"
BASE_PATH      = $(ROOT:/src/github.com/tigera/calico-k8sapiserver/=)
export GOPATH  = $(BASE_PATH):$(ROOT)/vendor

NON_VENDOR_DIRS = $(shell $(DOCKER_CMD) glide nv)

# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "apiserver" instead of "bin/apiserver".
#########################################################################
build: .generate_files \
       $(BINDIR)/calico-k8sapiserver

# We'll rebuild apiserver if any go file has changed (ie. NEWEST_GO_FILE)
$(BINDIR)/calico-k8sapiserver: .generate_files $(NEWEST_GO_FILE)
	$(GO_BUILD) -o $@ $(CAPI_PKG)/cmd/apiserver
	cp $(BINDIR)/calico-k8sapiserver $(ARTIFACTS)/simple-image/calico-k8sapiserver
	docker build -t calico-k8sapiserver:latest $(ARTIFACTS)/simple-image

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
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/defaulter-gen

$(BINDIR)/deepcopy-gen: 
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/deepcopy-gen

$(BINDIR)/conversion-gen: 
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/conversion-gen

$(BINDIR)/client-gen: 
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/client-gen

$(BINDIR)/lister-gen:
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/lister-gen

$(BINDIR)/informer-gen: 
	go build -o $@ $(CAPI_PKG)/vendor/k8s.io/kubernetes/cmd/libs/go2idl/informer-gen

$(BINDIR)/openapi-gen: vendor/k8s.io/kubernetes/cmd/libs/go2idl/openapi-gen
	go build -o $@ $(CAPI_PKG)/$^

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

# Some prereq stuff
###################

.init: $(cBuildImageTarget)
	touch $@

.cBuildImage: artifacts/simple-image/Dockerfile
	sed "s/GO_VERSION/$(GO_VERSION)/g" < artifacts/simple-image/Dockerfile | \
	  docker build -t cbuildimage -
	touch $@

# this target uses the host-local go installation to test 
test-integration: .init $(cBuildImageTarget) build
	# golang integration tests
	test/integration.sh

clean: clean-bin clean-build-image clean-generated
clean-bin:
	rm -rf $(BINDIR)
	rm -f .generate_exes

clean-build-image:
	docker rmi -f calico-k8sapiserver > /dev/null 2>&1 || true

clean-generated:
	rm -f .generate_files
	find $(TOP_SRC_DIRS) -name zz_generated* -exec rm {} \;
	# rollback changes to the generated clientset directories
	# find $(TOP_SRC_DIRS) -type d -name *_generated -exec rm -rf {} \;