# Copyright 2019-2024 Tigera Inc. All rights reserved.

GO_BUILD_VER ?= v0.90
ARCHES ?= amd64 arm64
SRC_DIR = $(pwd)
SRC_FILES = $(shell find $(SRC_DIR) -name '*.go')
TEST_DIR = $(pwd)

ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

GIT_USE_SSH=true

include ../Makefile.common

VERSION_FLAGS=-X main.VERSION=$(GIT_VERSION) \
			-X main.BUILD_DATE=$(DATE) \
			-X main.GIT_TAG=$(GIT_DESCRIPTION) \
			-X main.GIT_COMMIT=$(GIT_COMMIT)

# Target definitions.
eks-log-forwarder-startup: bin/eks-log-forwarder-startup-$(ARCH) bin/eks-log-forwarder-startup.exe
bin/eks-log-forwarder-startup-$(ARCH): $(SRC_FILES)
	@echo "Building eks init-container executable"
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) CGO_ENABLED=0 go build -o $@ -v -ldflags "$(VERSION_FLAGS) -s -w" $(SRC_DIR)'

bin/eks-log-forwarder-startup.exe: $(SRC_FILES)
	@echo "Building eks init-container executable for windows"
	$(DOCKER_GO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) CGO_ENABLED=0 GOOS=windows go build -o $@ -v -ldflags "$(VERSION_FLAGS) -s -w" $(SRC_DIR)'

.PHONY: ut
ut: eks-log-forwarder-startup
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) go test $(UNIT_TEST_FLAGS) $(TEST_DIR)'

.PHONY: clean
clean:
	rm -rf bin
