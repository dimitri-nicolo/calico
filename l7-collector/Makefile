include ../metadata.mk

PACKAGE_NAME          ?= github.com/projectcalico/calico/l7-collector

L7_COLLECTOR_IMAGE    ?=l7-collector
ENVOY_INIT_IMAGE      ?=envoy-init

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

SRC_FILES=$(shell find cmd pkg uds -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go')

VERSION_FLAGS=-X main.VERSION=$(GIT_VERSION) \
	-X main.BUILD_DATE=$(DATE) \
	-X main.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
	-X main.GIT_REVISION=$(GIT_COMMIT)

###############################################################################
# Building the binary
###############################################################################
build: bin/l7-collector-$(ARCH)

.PHONY: bin/l7-collector-$(ARCH)
bin/l7-collector-$(ARCH): $(SRC_FILES) protobuf
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/l7-collector/*.go'

# Generate the protobuf bindings for go. The proto/felixbackend.pb.go file is included in SRC_FILES
# In order to make use of complex structures like protobuf.google.timestamp, we need to link the
# protobuf google types when generating go code from protobuf messages
protobuf proto/felixbackend.pb.go: proto/felixbackend.proto
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		-v $(CURDIR):/code -v $(CURDIR)/proto:/src:rw \
		$(PROTOC_CONTAINER) \
		--gogofaster_out=\
	Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
	plugins=grpc:. \
	felixbackend.proto
	# Make sure the generated code won't cause a static-checks failure.
	$(MAKE) fix

###############################################################################
# Building the image
###############################################################################
L7_COLLECTOR_CONTAINER_CREATED=.l7-collector.created-$(ARCH)
ENVOY_INIT_CONTAINER_CREATED=.envoy-init.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(L7_COLLECTOR_IMAGE) $(ENVOY_INIT_IMAGE)

$(L7_COLLECTOR_IMAGE): $(L7_COLLECTOR_CONTAINER_CREATED)
$(L7_COLLECTOR_CONTAINER_CREATED): Dockerfile.$(ARCH) bin/l7-collector-$(ARCH)
	$(DOCKER_BUILD) -t $(L7_COLLECTOR_IMAGE):latest-$(ARCH) -f Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(L7_COLLECTOR_IMAGE)
	touch $@

$(ENVOY_INIT_IMAGE): $(ENVOY_INIT_CONTAINER_CREATED)
$(ENVOY_INIT_CONTAINER_CREATED): envoy-init/Dockerfile.$(ARCH) envoy-init/init_iptables.sh
	$(DOCKER_BUILD) -t $(ENVOY_INIT_IMAGE):latest-$(ARCH) -f envoy-init/Dockerfile.$(ARCH) envoy-init --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest BUILD_IMAGES=$(ENVOY_INIT_IMAGE)
	touch $@

###############################################################################
# UTs
###############################################################################
.PHONY: ut
ut: bin/l7-collector-$(ARCH)
	$(DOCKER_GO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo -r --skipPackage deps,fv -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: fv
fv: bin/l7-collector-$(ARCH)
	$(DOCKER_RUN) \
	    $(LOCAL_BUILD_MOUNTS) \
	    $(CALICO_BUILD) sh -c "$(GIT_CONFIG_SSH) \
	    ginkgo fv -r --skipPackage deps -focus='$(GINKGO_FOCUS)' $(GINKGO_ARGS) $(WHAT)"

.PHONY: clean
clean:
	rm -rf bin proto/felixbackend.pb.go
	rm -f $(L7_COLLECTOR_CONTAINER_CREATED)
	rm -f $(ENVOY_INIT_CONTAINER_CREATED)

	-docker image rm -f $$(docker images $(L7_COLLECTOR_IMAGE) -a -q)
	-docker image rm -f $$(docker images $(ENVOY_INIT_IMAGE) -a -q)

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

ci: clean static-checks ut

cd: image-all cd-common
