PACKAGE_NAME ?= github.com/tigera/prometheus-service
GO_BUILD_VER ?= v0.53
GIT_USE_SSH = true
ORGANIZATION = tigera

PROMETHEUS_SERVICE_IMAGE ?=tigera/prometheus-service
BUILD_IMAGES ?=$(PROMETHEUS_SERVICE_IMAGE)
##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

build: prometheus-service
BINDIR?=bin
BUILD_DIR?=build
TOP_SRC_DIRS=pkg
GO_FILES= $(shell sh -c "find pkg cmd -name \\*.go")

PROMETHEUS_SERVICE_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
PROMETHEUS_SERVICE_BUILD_DATE?=$(shell date -u +'%FT%T%z')
PROMETHEUS_SERVICE_GIT_COMMIT?=$(shell git rev-parse --short HEAD)
PROMETHEUS_SERVICE_GIT_TAG?=$(shell git describe --tags)

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/handler.VERSION=$(PROMETHEUS_SERVICE_VERSION) \
	-X $(PACKAGE_NAME)/pkg/handler.BUILD_DATE=$(PROMETHEUS_SERVICE_BUILD_DATE) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_TAG=$(PROMETHEUS_SERVICE_GIT_TAG) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_COMMIT=$(PROMETHEUS_SERVICE_GIT_COMMIT) \
	-X main.VERSION=$(PROMETHEUS_SERVICE_VERSION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "prometheus-service" instead of "$(BINDIR)/prometheus-service".
prometheus-service: $(BINDIR)/prometheus-service

$(BINDIR)/prometheus-service: $(BINDIR)/prometheus-service-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd $(BINDIR) && ln -s -T prometheus-service-$(ARCH) prometheus-service'

$(BINDIR)/prometheus-service-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) "$(PACKAGE_NAME)/cmd/server" && \
				( ldd $(BINDIR)/prometheus-service-$(ARCH) 2>&1 | \
	                grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $(BINDIR)/prometheus-service-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(BUILD_IMAGES) $(addsuffix -$(ARCH),$(BUILD_IMAGES))

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(PROMETHEUS_SERVICE_IMAGE)
$(PROMETHEUS_SERVICE_IMAGE): $(PROMETHEUS_SERVICE_IMAGE)-$(ARCH)
$(PROMETHEUS_SERVICE_IMAGE)-$(ARCH): $(BINDIR)/prometheus-service-$(ARCH)
	docker build --pull -t $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) -f ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) $(PROMETHEUS_SERVICE_IMAGE):latest
endif

.PHONY: clean
clean:
	docker rmi -f $(PROMETHEUS_SERVICE_IMAGE):latest > /dev/null 2>&1
	docker rmi -f $(PROMETHEUS_SERVICE_IMAGE):latest-$(ARCH) > /dev/null 2>&1
	rm -rf $(BINDIR) .go-pkg-cache Makefile.common*

###############################################################################
# Utils
###############################################################################
# this is not a linked target, available for convenience.
.PHONY: tidy
## 'tidy' go modules.
tidy:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod tidy'
