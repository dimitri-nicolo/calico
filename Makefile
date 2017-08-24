.PHONY: all test

default: all
all: test
test: ut

BUILD_VER:=latest
BUILD_IMAGE:=calico/calicoq
PACKAGE_NAME?=github.com/tigera/calicoq
LOCAL_USER_ID?=$(shell id -u $$USER)
BINARY:=bin/calicoq

GO_BUILD_VER?=latest
GO_BUILD?=calico/go-build:$(GO_BUILD_VER)

.PHONY: vendor
vendor:
	glide install --strip-vendor

.PHONY: update-vendor
update-vendor:
	glide up --strip-vendor

.PHONY: ut
ut: bin/calicoq
	ginkgo -cover -r --skipPackage vendor calicoq/*

	@echo
	@echo '+==============+'
	@echo '| All coverage |'
	@echo '+==============+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_

	@echo
	@echo '+==================+'
	@echo '| Missing coverage |'
	@echo '+==================+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_ | grep -v '100.0%'

.PHONY: ut-containerized
ut-containerized: bin/calicoq
	docker run --rm -t \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		calico/go-build \
		sh -c 'make ut'

.PHONY: fv
fv: bin/calicoq
	CALICOQ=`pwd`/$^ fv/run-test

.PHONY: fv-containerized
fv-containerized: bin/calicoq build-image
	docker run --net=host --privileged \
		--rm -t \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(BUILD_IMAGE) \
		sh -c 'CALICOQ=`pwd`/$(BINARY) fv/run-test'
	$(MAKE) clean-image

.PHONY: st
st: bin/calicoq
	CALICOQ=`pwd`/$^ st/run-test

.PHONY: st-containerized
st-containerized: bin/calicoq build-image
	docker run --net=host --privileged \
		--rm -t \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(BUILD_IMAGE) \
		sh -c 'CALICOQ=`pwd`/$(BINARY) st/run-test'
	$(MAKE) clean-image

# Build image for containerized testing
.PHONY: build-image
build-image:
	docker build -t $(BUILD_IMAGE):$(BUILD_VER) `pwd`

# Clean up image from containerized testing
.PHONY: clean-image
clean-image:
	docker rmi $(shell docker images -a | grep $(BUILD_IMAGE) | awk '{print $$3}')

# All calicoq Go source files.
CALICOQ_GO_FILES:=$(shell find calicoq -type f -name '*.go' -print)

bin/calicoq:
	$(MAKE) binary-containerized

.PHONY: binary-containerized
binary-containerized: $(CALICOQ_GO_FILES)
	mkdir -p bin
	mkdir -p $(HOME)/.glide
	# vendor in a container first
	docker run --rm \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-v $(HOME)/.glide:/home/user/.glide:rw \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		calico/go-build \
		sh -c 'glide install --strip-vendor'
	# Generate the protobuf bindings for Felix
	# Cannot do this together with vendoring since docker permissions in go-build are not perfect
	$(MAKE) vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go
	# Create the binary
	docker run --rm -t \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		calico/go-build \
		sh -c 'go build -o "$(BINARY)" "./calicoq/calicoq.go"'

.PHONY: binary
binary: vendor vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go $(CALICOQ_GO_FILES)
	mkdir -p bin
	go build -o "$(BINARY)" "./calicoq/calicoq.go"

release/calicoq: $(CALICOQ_GO_FILES)
	mkdir -p release
	cd build-container && docker build -t calicoq-build .
	docker run --rm -v `pwd`:/calicoq calicoq-build /calicoq/build-container/build.sh

# Generate the protobuf bindings for Felix.
vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go: vendor/github.com/projectcalico/felix/proto/felixbackend.proto
	docker run --rm -v `pwd`/vendor/github.com/projectcalico/felix/proto:/src:rw \
	              calico/protoc \
	              --gogofaster_out=. \
	              felixbackend.proto

.PHONY: clean
clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release vendor
	-docker rm -f calico-build
	-docker rmi calico/build
	-docker rmi $(BUILD_IMAGE)
	-docker rmi calico/go-build
