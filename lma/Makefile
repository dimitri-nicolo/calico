include ../metadata.mk

PACKAGE_NAME    ?= github.com/projectcalico/calico/lma

RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

build: ut

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/api/ReportEventFetcher \
	pkg/api/ListDestination \
	pkg/elastic/ClusterContextClientFactory \
	pkg/elastic/FlowFilter \
	pkg/elastic/Client \
	pkg/auth/RBACAuthorizer \
	pkg/rbac/FlowHelper \
	pkg/k8s/ClientSet \
	pkg/k8s/ClientSetFactory \
	pkg/auth/JWTAuth

# Note that mockery doesn't add the correct elasticsearch dependency, so if you uncomment this to regenerate the client
# be sure to check that the elasticsearch dependency for the generated mock is correct
#MOCKERY_FILE_PATHS+= \
#	pkg/elastic/Client

##############################################################################
TOP_SRC_DIRS     = pkg
TEST_DIRS       ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                        -exec dirname {} \\; | sort | uniq")

ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

#############################################################################
# Testing
#############################################################################
report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir run-elastic
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'
	-docker rm -f tigera-elastic

###############################################################################
# CI
###############################################################################
.PHONY: ci

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
ci: static-checks ut
