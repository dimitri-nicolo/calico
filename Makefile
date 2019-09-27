# Shortcut targets
default: ut 

all: ut

## Run the tests for the current platform/architecture
test: ut

# Define some constants
#############################
ELASTIC_VERSION ?= 7.3.2

###############################################################################
# Both native and cross architecture builds are supported.
# The target architecture is select by setting the ARCH variable.
# When ARCH is undefined it is set to the detected host architecture.
# When ARCH differs from the host architecture a crossbuild will be performed.
ARCHES=$(patsubst Dockerfile.%,%,$(wildcard Dockerfile.*))

# BUILDARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
BUILDARCH ?= $(shell uname -m)
BUILDOS ?= $(shell uname -s | tr A-Z a-z)

# canonicalized names for host architecture
ifeq ($(BUILDARCH),aarch64)
        BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
        BUILDARCH=amd64
endif

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
ARCH ?= $(BUILDARCH)

# canonicalized names for target architecture
ifeq ($(ARCH),aarch64)
        override ARCH=arm64
endif
ifeq ($(ARCH),x86_64)
    override ARCH=amd64
endif

# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

# these macros create a list of valid architectures for pushing manifests
space :=
space +=
comma := ,
prefix_linux = $(addprefix linux/,$(strip $1))
join_platforms = $(subst $(space),$(comma),$(call prefix_linux,$(strip $1)))

###############################################################################
BUILD_IMAGE?=tigera/lma
PUSH_IMAGES?=gcr.io/unique-caldron-775/cnx/$(BUILD_IMAGE)
RELEASE_IMAGES?=quay.io/$(BUILD_IMAGE)
PACKAGE_NAME?=github.com/tigera/lma

# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
EXCLUDE_MANIFEST_REGISTRIES ?= quay.io/
PUSH_MANIFEST_IMAGES=$(PUSH_IMAGES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGES=$(filter-out $(PUSH_MANIFEST_IMAGES),$(PUSH_IMAGES))

# location of docker credentials to push manifests
DOCKER_CONFIG ?= $(HOME)/.docker/config.json

GO_BUILD_VER?=v0.24
# For building, we use the go-build image for the *host* architecture, even if the target is different
# the one for the host should contain all the necessary cross-compilation tools
# we do not need to use the arch since go-build:v0.15 now is multi-arch manifest
CALICO_BUILD=calico/go-build:$(GO_BUILD_VER)

ETCD_VERSION?=v3.3.7
# If building on amd64 omit the arch in the container name.  Fixme!
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)

# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

# Some env vars that devs might find useful:
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Define some constants
#######################
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

ifdef LMA_PATH
  EXTRA_DOCKER_ARGS += -v $(LMA_PATH):/go/src/github.com/tigera/lma:ro
endif

# SSH_AUTH_SOCK doesn't work with MacOS but we can optionally volume mount keys 
ifdef SSH_AUTH_DIR
	EXTRA_DOCKER_ARGS += --tmpfs /home/user -v $(SSH_AUTH_DIR):/home/user/.ssh:ro  
endif

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif


DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   mkdir -p .go-build-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -e GOARCH=$(ARCH) \
                              -v $(CURDIR):/$(PACKAGE_NAME):rw \
                              -v $(CURDIR)/.go-pkg-cache:/go/pkg:rw \
                              -v $(CURDIR)/report:/report:rw \
                              -w /$(PACKAGE_NAME) \
                              $(CALICO_BUILD)

##########################################################################
# Testing
##########################################################################
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

###############################################################################
# Utilities
###############################################################################
.PHONY: help
## Display this help text
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	$(info Available targets)
	@awk '/^[a-zA-Z\-\_0-9\/]+:/ {                                      \
		nb = sub( /^## /, "", helpMsg );                                \
		if(nb == 0) {                                                   \
			helpMsg = $$0;                                              \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                   \
		}                                                               \
		if (nb)                                                         \
			printf "\033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;  \
	}                                                                   \
	{ helpMsg = $$0 }'                                                  \
	width=20                                                            \
	$(MAKEFILE_LIST)

LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')

