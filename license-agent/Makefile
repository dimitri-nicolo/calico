include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/license-agent

build: license-agent

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_LICENSE_AGENT_PROJECT_ID)

LICENSE_AGENT_IMAGE        ?=tigera/license-agent
BUILD_IMAGES               ?=$(LICENSE_AGENT_IMAGE)

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

K8S_VERSION    = v1.22.0
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go") \
								 		$(shell find ../libcalico-go/lib -name '*.go') \
										$(shell find ../licensing/client -name '*.go')

VERSION_FLAGS = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
								-X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
								-X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
								-X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "license-agent" instead of "$(BINDIR)/license-agent".
license-agent: $(BINDIR)/license-agent

$(BINDIR)/license-agent: $(BINDIR)/license-agent-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd $(BINDIR) && ln -s -T license-agent-$(ARCH) license-agent'

$(BINDIR)/license-agent-$(ARCH): $(GO_FILES)
	@echo Building license-agent...
	mkdir -p .go-build-cache && \
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" "$(PACKAGE_NAME)/cmd/server"'

# Build the docker image.
.PHONY: $(LICENSE_AGENT_IMAGE)-$(ARCH)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

LICENSE_AGENT_CONTAINER_CREATED=.$(LICENSE_AGENT_IMAGE).created-$(ARCH)

image: $(LICENSE_AGENT_IMAGE)
$(LICENSE_AGENT_IMAGE): $(LICENSE_AGENT_CONTAINER_CREATED)
$(LICENSE_AGENT_CONTAINER_CREATED): $(BINDIR)/license-agent-$(ARCH)
	$(DOCKER_BUILD) --pull -t $(LICENSE_AGENT_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

##########################################################################
# Testing
##########################################################################
report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go test $(UNIT_TEST_FLAGS) $(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: clean
clean:
	rm -rf $(BINDIR) bin
	-docker image rm -f $$(docker images $(LICENSE_AGENT_IMAGE) -a -q)

###############################################################################
# Static checks
###############################################################################

###############################################################################
# See .golangci.yml for golangci-lint config
LINT_ARGS +=

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean image-all static-checks ut

## Deploys images to registry
cd: image-all cd-common
