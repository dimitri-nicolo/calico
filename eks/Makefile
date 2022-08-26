# Copyright 2019-2022 Tigera Inc. All rights reserved.

all: eks-log-forwarder-startup

# This makefile applies only to non-Windows systems.
# For Windows, we override the main target to do nothing.
ifeq ($(OS),Windows_NT)
eks-log-forwarder-startup:

else

GO_BUILD_VER    ?= v0.73
CALICO_BUILD     = calico/go-build:$(GO_BUILD_VER)
BINDIR          ?= bin
SRC_DIR          = $(pwd)
SRC_FILES        = $(shell find $(SRC_DIR) -name '*.go')
TEST_DIR         = $(pwd)
ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

VERSION?=$(shell git describe --tags --dirty --always)
BUILD_DATE?=$(shell date -u +'%FT%T%z')
GIT_COMMIT?=$(shell git rev-parse --short HEAD)
GIT_TAG?=$(shell git describe --tags)

VERSION_FLAGS=-X main.VERSION=$(VERSION) \
			-X main.BUILD_DATE=$(BUILD_DATE) \
			-X main.GIT_TAG=$(GIT_TAG) \
			-X main.GIT_COMMIT=$(GIT_COMMIT)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
LOCAL_UID:=$(shell id -u)
LOCAL_GID:=$(shell id -g)

ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif

MY_DOCKER_GID=$(shell getent group docker | cut -d: -f3)

DOCKER_GO_BUILD = mkdir -p .go-pkg-cache && \
                  mkdir -p .go-build-cache && \
                  docker run --rm \
                        --net=host \
                        -e LOCAL_USER_ID=$(LOCAL_UID) \
                        -e LOCAL_GROUP_ID=$(LOCAL_GID) \
                        -e GOARCH=$(ARCH) \
                        -v $${PWD}/:/eks/:rw \
                        -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
                        -v $${PWD}/.go-build-cache:/go-build:rw \
                        -v /var/run/docker.sock:/var/run/docker.sock:rw \
                        -e GOCACHE=/go-build \
                        -e EXTRA_GROUP_ID=$(MY_DOCKER_GID) \
                        -w /eks \
                        $(EXTRA_DOCKER_ARGS) \
                        $(CALICO_BUILD)

# Target definitions.
eks-log-forwarder-startup: bin/eks-log-forwarder-startup bin/eks-log-forwarder-startup.exe
bin/eks-log-forwarder-startup: $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	@echo "Building eks init-container executable"
	mkdir -p $(BINDIR)
	$(DOCKER_GO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	        go build -o $@ -v $(LDFLAGS) $(SRC_DIR) && \
	            ( ldd $@ 2>&1 | grep -q "Not a valid dynamic program\|not a dynamic executable" || \
	                ( echo "Error: $(BINDIR)/$@ was not statically linked"; false ) )'

bin/eks-log-forwarder-startup.exe: $(SRC_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	$(eval EXTRA_DOCKER_ARGS += -e GOOS=windows)
	@echo "Building eks init-container executable for windows"
	mkdir -p $(BINDIR)
	$(DOCKER_GO_BUILD) \
	    sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	        go build -o $@ -v $(LDFLAGS) $(SRC_DIR) && \
	            ( ldd $@ 2>&1 | grep -q "Not a valid dynamic program\|not a dynamic executable" || \
	                ( echo "Error: $(BINDIR)/$@ was not statically linked"; false ) )'

.PHONY: ut
ut: test-eks-log-forwarder-startup
test-eks-log-forwarder-startup: eks-log-forwarder-startup
	$(DOCKER_GO_BUILD) \
	sh -c 'git config --global url."git@github.com:tigera".insteadOf "https://github.com/tigera" && \
	    go test $(UNIT_TEST_FLAGS) $(TEST_DIR)'

.PHONY: clean
clean: clean-eks-log-forwarder-startup
clean-eks-log-forwarder-startup:
	rm -rf $(BINDIR)

endif
