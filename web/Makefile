.PHONY: all test

default: all
all: test
test: ut

# The build architecture is select by setting the ARCH variable.
# For example: When building on ppc64le you could use ARCH=ppc64le make <....>.
# When ARCH is undefined it defaults to amd64.
ARCH?=amd64
ifeq ($(ARCH),amd64)
	ARCHTAG?=
	GO_BUILD_VER?=v0.9
endif

ifeq ($(ARCH),ppc64le)
	ARCHTAG:=-ppc64le
	GO_BUILD_VER?=latest
endif

GO_BUILD_CONTAINER?=calico/go-build$(ARCHTAG):$(GO_BUILD_VER)

# The package name (actually the sub-directory within that package)
PACKAGE_NAME?=github.com/tigera/calicoq/web

# The container image name
BUILD_IMAGE:=tigera/cnx-queryserver

# The binary image name
BINARY:=bin/cnx-queryserver

# Local environment
LOCAL_USER_ID?=$(shell id -u $$USER)
QUERYSERVER_VERSION?=$(shell git describe --tags --dirty --always)
QUERYSERVER_BUILD_DATE?=$(shell date -u +'%FT%T%z')
QUERYSERVER_GIT_REVISION?=$(shell git rev-parse --short HEAD)
QUERYSERVER_GIT_DESCRIPTION?=$(shell git describe --tags)

VERSION_FLAGS=-X $(PACKAGE_NAME)/queryserver/handlers.VERSION=$(QUERYSERVER_VERSION) \
	-X $(PACKAGE_NAME)/queryserver/handlers.BUILD_DATE=$(QUERYSERVER_BUILD_DATE) \
	-X $(PACKAGE_NAME)/queryserver/handlers.GIT_DESCRIPTION=$(QUERYSERVER_GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/queryserver/handlers.GIT_REVISION=$(QUERYSERVER_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

.PHONY: vendor
vendor vendor/.up-to-date: glide.lock
	mkdir -p $(HOME)/.glide
	docker run --rm \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-v $(HOME)/.glide:/home/user/.glide:rw \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		$(GO_BUILD_CONTAINER) \
		sh -c 'glide install --strip-vendor'
	touch vendor/.up-to-date

.PHONY: update-vendor
update-vendor:
	glide up --strip-vendor

.PHONY: ut
ut: vendor/.up-to-date
	./run-uts

.PHONY: fv
fv: vendor/.up-to-date
	./run-fvs

.PHONY: ut-containerized
fv-containerized: vendor/.up-to-date run-etcd vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go
	-mkdir -p .go-pkg-cache
	docker run --rm --privileged --net=host \
    -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
    -v $(CURDIR)/.go-pkg-cache:/go/pkg/:rw \
    -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
    $(GO_BUILD_CONTAINER) sh -c 'cd /go/src/$(PACKAGE_NAME) && make WHAT=$(WHAT) SKIP=$(SKIP) fv'

.PHONY: fv-containerized
ut-containerized:
	@echo "Need to add tests. This is a no-op"

.PHONY: st-containerized
st-containerized:
	@echo "Need to add tests. This is a no-op"

# Build image for containerized testing
.PHONY: build-image tigera/cnx-queryserver
tigera/cnx-queryserver build-image .container-created: bin/cnx-queryserver
	docker build -t $(BUILD_IMAGE) `pwd`
	touch .container-created

# Clean up image from containerized testing
.PHONY: clean-image
clean-image:
	docker rmi -f $(shell docker images -a | grep $(BUILD_IMAGE) | awk '{print $$3}' | awk '!a[$$0]++')

# All queryserver Go source files.
QUERYSERVER_GO_FILES:=$(shell find queryserver pkg -type f -name '*.go' -print)

bin/cnx-queryserver: $(QUERYSERVER_GO_FILES) vendor/.up-to-date vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go
	$(MAKE) binary-containerized

.PHONY: binary-containerized
binary-containerized:
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	# Create the binary
	mkdir -p bin
	docker run --rm -t \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-v $$SSH_AUTH_SOCK:/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		$(GO_BUILD_CONTAINER) \
		sh -c 'go build $(LDFLAGS) -o "$(BINARY)" "./queryserver/queryserver.go"'

.PHONY: binary
binary: vendor vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go $(QUERYSERVER_GO_FILES)
	mkdir -p bin
	go build $(BUILD_LDFLAGS) -o "$(BINARY)" "./queryserver/queryserver.go"

# Generate the protobuf bindings for Felix.
vendor/github.com/projectcalico/felix/proto/felixbackend.pb.go: vendor/github.com/projectcalico/felix/proto/felixbackend.proto
	docker run --rm -v `pwd`/vendor/github.com/projectcalico/felix/proto:/src:rw \
	              calico/protoc \
	              --gogofaster_out=. \
	              felixbackend.proto

.PHONY: clean-release
clean-release:
	-rm -rf release

.PHONY: clean
clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release vendor
	-docker rmi calico/build:latest
	-docker rmi $(BUILD_IMAGE) -f
	-docker rmi calico/go-build -f

## Run etcd as a container (calico-etcd)
run-etcd: stop-etcd
	docker run --detach \
	--net=host \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:v3.1.7 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stop the etcd container (calico-etcd)
stop-etcd:
	-docker rm -f calico-etcd

run-cnx-queryserver: stop-cnx-queryserver .container-created
	docker run --detach \
	--net=host \
	--name cnx-queryserver \
	-e DATASTORE_TYPE=etcdv3 \
	-e ETCD_ENDPOINTS=http://localhost:2379 \
	$(BUILD_IMAGE)

## Stop the cnx-queryserver container (calico-etcd)
stop-cnx-queryserver:
	-docker rm -f cnx-queryserver

testenv: run-etcd run-cnx-queryserver .testenv-container-created
	-docker run --net=host --privileged \
	           --uts=host \
	           --pid=host \
	           --rm -ti \
	           -e DATASTORE_TYPE=etcdv3 \
	           -e ETCD_ENDPOINTS=http://127.0.0.1:2379 \
	           -v `pwd`:/code \
               -v /var/run/docker.sock:/var/run/docker.sock \
	           --name=testenv \
	           tigera/cnx-queryserver-testenv \
	           sh

# Build image for containerized testing
.PHONY: testenv-build-image
testenv-build-image .testenv-container-created:
	docker build -f Dockerfile.testenv -t tigera/cnx-queryserver-testenv `pwd`
	touch .testenv-container-created
