PACKAGE_NAME?=github.com/tigera/licensing
GO_BUILD_VER?=v0.30

###############################################################################
# Download and include Makefile.common before anything else
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

###############################################################################

BUILD_IMAGE?=calico/carrotctl
#PUSH_IMAGES?=$(BUILD_IMAGE) quay.io/calico/carrotctl
PUSH_IMAGES?=$(BUILD_IMAGE)
RELEASE_IMAGES?=
CARROTCTL_DIR=carrotctl
SRC_FILES=$(shell find $(CARROTCTL_DIR) -name '*.go')

GO_BUILD_CONTAINER?=calico/go-build:$(GO_BUILD_VER)-$(BUILDARCH)

CARROTCTL_VERSION?=$(shell git describe --tags --dirty --always)
CARROTCTL_GIT_REVISION?=$(shell git rev-parse --short HEAD)
CARROTCTL_BUILD_DATE?=$(shell date -u +'%FT%T%z')
LDFLAGS=-ldflags "-X $(PACKAGE_NAME)/carrotctl/cmd.VERSION=$(CARROTCTL_VERSION) \
	-X $(PACKAGE_NAME)/carrotctl/cmd.BUILD_DATE=$(CARROTCTL_BUILD_DATE) \
	-X $(PACKAGE_NAME)/carrotctl/cmd.GIT_REVISION=$(CARROTCTL_GIT_REVISION) -s -w"

.PHONY: clean
## Clean enough that a new release build will be clean
clean:
	find . -name '*.created-$(ARCH)' -exec rm -f {} +
	rm -rf bin build certs *.tar report/
	docker rmi $(BUILD_IMAGE):latest-$(ARCH) || true
	docker rmi $(BUILD_IMAGE):$(VERSION)-$(ARCH) || true
ifeq ($(ARCH),amd64)
	docker rmi $(BUILD_IMAGE):latest || true
	docker rmi $(BUILD_IMAGE):$(VERSION) || true
endif

###############################################################################
# Building the binary
###############################################################################
LOCAL_DOCKER_ARGS := -i -e ARCH=$(ARCH) -e CARROTCTL_VERSION=$(CARROTCTL_VERSION) \
	                -e CARROTCTL_BUILD_DATE=$(CARROTCTL_BUILD_DATE) \
	                -e CARROTCTL_GIT_REVISION=$(CARROTCTL_GIT_REVISION) \
	                -v $(CURDIR)/bin:/go/src/$(PACKAGE_NAME)/bin

.PHONY: build-all
## Build the binaries for all architectures and platforms
build-all: $(addprefix bin/carrotctl-linux-,$(VALIDARCHES)) bin/carrotctl-windows-amd64.exe bin/carrotctl-darwin-amd64

.PHONY: build
## Build the binary for the current architecture and platform
build: bin/carrotctl-$(BUILDOS)-$(ARCH)

# The supported different binary names. For each, ensure that an BUILDOS and ARCH is set
bin/carrotctl-%-amd64: ARCH=amd64
bin/carrotctl-%-arm64: ARCH=arm64
bin/carrotctl-%-ppc64le: ARCH=ppc64le
bin/carrotctl-%-s390x: ARCH=s390x
bin/carrotctl-darwin-amd64: BUILDOS=darwin
bin/carrotctl-windows-amd64: BUILDOS=windows
bin/carrotctl-linux-%: BUILDOS=linux

bin/carrotctl-%: $(SRC_FILES)
	$(DOCKER_RUN) $(LOCAL_DOCKER_ARGS) $(CALICO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && GOPRIVATE="github.com/tigera"\
	    go build -v -o bin/carrotctl-$(BUILDOS)-$(ARCH) $(LDFLAGS) "$(CARROTCTL_DIR)/carrotctl.go"'

.PHONY: calico/carrotctl
# Overrides for the binaries that need different output names
bin/carrotctl: bin/carrotctl-linux-amd64
	cp $< $@
bin/carrotctl-windows-amd64.exe: bin/carrotctl-windows-amd64
	mv $< $@

###############################################################################
# UTs
###############################################################################
.PHONY: ut
## Run the tests in a container. Useful for CI, Mac dev.
ut: $(SRC_FILES)
	mkdir -p report
	$(DOCKER_GO_BUILD) /bin/bash -c "go test -v ./... | go-junit-report > ./report/tests.xml"

fv st:
	@echo "No FVs or STs available"
