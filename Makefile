PACKAGE_NAME    ?= github.com/tigera/lma
GO_BUILD_VER    ?= v0.51
GIT_USE_SSH     := true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private
APISERVER_REPO   = github.com/tigera/apiserver
LOCAL_CHECKS     = mod-download

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_LMA_PROJECT_ID)

# Used so semaphore can trigger the update pin pipelines in projects that have this project as a dependency.
SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_COMPLIANCE_PROJECT_ID) $(SEMAPHORE_ES_PROXY_IMAGE_PROJECT_ID) \
	$(SEMAPHORE_VOLTRON_PROJECT_ID)

build: ut

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH ?= $(GO_BUILD_VER)
MAKE_REPO   ?= https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/api/EventFetcher \
	pkg/api/ListDestination \
	pkg/elastic/ClusterContextClientFactory \
	pkg/elastic/FlowFilter \
	pkg/auth/RBACAuthorizer \
	pkg/rbac/FlowHelper

# Note that mockery doesn't add the correct elasticsearch dependency, so if you uncomment this to regenerate the client
# be sure to check that the elasticsearch dependency for the generated mock is correct
#MOCKERY_FILE_PATHS+= \
#	pkg/elastic/Client

ifdef LMA_PATH
EXTRA_DOCKER_ARGS += -v $(LMA_PATH):/go/src/github.com/tigera/lma:ro
endif

# SSH_AUTH_SOCK doesn't work with MacOS but we can optionally volume mount keys
ifdef SSH_AUTH_DIR
EXTRA_DOCKER_ARGS += --tmpfs /home/user -v $(SSH_AUTH_DIR):/home/user/.ssh:ro
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

include Makefile.common

##############################################################################
ELASTIC_VERSION ?= 7.3.2

BUILD_IMAGE     ?= tigera/lma
PUSH_IMAGES     ?= gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES  ?= quay.io/$(BUILD_IMAGE)

TOP_SRC_DIRS     = pkg
SRC_DIRS         = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                        -exec dirname {} \\; | sort | uniq")
TEST_DIRS       ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                        -exec dirname {} \\; | sort | uniq")
GO_FILES         = $(shell sh -c "find pkg cmd -name \\*.go")

ifeq ($(shell uname -s),Darwin)
STAT = stat -f '%c %N'
else
STAT = stat -c '%Y %n'
endif

ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

##############################################################################
# Updating pins
##############################################################################
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin replace-apiserver-pin

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

## Run elasticsearch as a container (tigera-elastic)
run-elastic: stop-elastic
	# Run ES on Docker.
	docker run --detach \
	--net=host \
	--name=tigera-elastic \
	-e "discovery.type=single-node" \
	docker.elastic.co/elasticsearch/elasticsearch:$(ELASTIC_VERSION)

	# Wait until ES is accepting requests.
	@while ! docker exec tigera-elastic curl localhost:9200 2> /dev/null; do echo "Waiting for Elasticsearch to come up..."; sleep 2; done


## Stop elasticsearch with name tigera-elastic
stop-elastic:
	-docker rm -f tigera-elastic

###############################################################################
# CI/
###############################################################################
.PHONY: ci

###############################################################################
# See .golangci.yml for golangci-lint config
LINT_ARGS +=

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
ci: static-checks ut

###############################################################################
# Utils
###############################################################################
# this is not a linked target, available for convenience.
.PHONY: tidy
## 'tidy' go modules.
tidy:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod tidy'
