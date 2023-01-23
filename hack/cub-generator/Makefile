include ../../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/hack/cub-generator

LOCAL_CHECKS = goimports check-gen-files

###############################################################################
# Download and include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../../lib.Makefile

###############################################################################

###############################################################################
# Env vars related to building
###############################################################################
SRC_FILES = $(shell find pkg cmd -name '*.go')

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(GIT_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(GIT_COMMIT)

# We need CGO to leverage Boring SSL.  However, the cross-compile doesn't support CGO yet.
ifeq ($(ARCH), $(filter $(ARCH),amd64))
CGO_ENABLED=1
else
CGO_ENABLED=0
endif

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/cub-generator-$(ARCH)

.PHONY: bin/cub-generator-$(ARCH)
bin/cub-generator-$(ARCH): $(SRC_FILES)
		$(DOCKER_GO_BUILD_CGO) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/$*/*.go'


###############################################################################
# Tests
###############################################################################
WHAT?=.
GINKGO_FOCUS?=.*

.PHONY:ut
## Run the fast set of unit tests in a container.
ut:
	$(DOCKER_RUN) --privileged $(CALICO_BUILD) \
		sh -c 'cd /go/src/$(PACKAGE_NAME) && ginkgo -r -focus="$(GINKGO_FOCUS)" $(WHAT)'

.PHONY:fv
fv:
	@echo "No FVs available"

st:
	@echo "No STs available"

###############################################################################
# CI
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean static-checks test
