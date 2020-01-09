PACKAGE_NAME    ?= github.com/tigera/lma
GO_BUILD_VER    ?= v0.32
GIT_USE_SSH     := true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private

build: ut

##############################################################################
# Download and include Makefile.common before anything else
##############################################################################
MAKE_BRANCH ?= $(GO_BUILD_VER)
MAKE_REPO   ?= https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

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

ifdef LMA_PATH
	EXTRA_DOCKER_ARGS += -v $(LMA_PATH):/go/src/github.com/tigera/lma:ro
endif

# SSH_AUTH_SOCK doesn't work with MacOS but we can optionally volume mount keys
ifdef SSH_AUTH_DIR
	EXTRA_DOCKER_ARGS += --tmpfs /home/user -v $(SSH_AUTH_DIR):/home/user/.ssh:ro
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

##############################################################################
# Updating pins
##############################################################################
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin

#############################################################################
# Testing
#############################################################################
report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir run-elastic
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url.ssh://git@github.com.insteadOf https://github.com && \
			go test $(UNIT_TEST_FLAGS) \
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

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
ci: ut
