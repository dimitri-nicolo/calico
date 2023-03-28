include ../metadata.mk
include ../lib.Makefile

PACKAGE_NAME ?= github.com/projectcalico/calico/selinux

SRC_FILES = calico-selinux.spec \
	calico.fc \
	calico.if \
	calico.te

###############################################################################
# Build SELinux builder image
###############################################################################
SELINUX_BUILD_IMAGE_EL8 =calico/selinux-build.el8
SELINUX_BUILD_CONTAINER_CREATED_EL8=.selinux-build.el8.created

SELINUX_BUILD_IMAGE_EL9 =calico/selinux-build.el9
SELINUX_BUILD_CONTAINER_CREATED_EL9=.selinux-build.el9.created

.PHONY: selinux-build.el8 selinux-build.el9

selinux-build.el8: $(SELINUX_BUILD_IMAGE_EL8)
$(SELINUX_BUILD_IMAGE_EL8): $(SELINUX_BUILD_CONTAINER_CREATED_EL8)
$(SELINUX_BUILD_CONTAINER_CREATED_EL8): Dockerfile.el8
	$(DOCKER_BUILD) -t $(SELINUX_BUILD_IMAGE_EL8):latest -f Dockerfile.el8 . --load
	touch $@

selinux-build.el9: $(SELINUX_BUILD_IMAGE_EL9)
$(SELINUX_BUILD_IMAGE_EL9): $(SELINUX_BUILD_CONTAINER_CREATED_EL9)
$(SELINUX_BUILD_CONTAINER_CREATED_EL9): Dockerfile.el9
	$(DOCKER_BUILD) -t $(SELINUX_BUILD_IMAGE_EL9):latest -f Dockerfile.el9 . --load
	touch $@

###############################################################################
# Build RPM
###############################################################################
.PHONY: build
build: calico-selinux.el8.rpm calico-selinux.el9.rpm

.PHONY: calico-selinux.el8.rpm
calico-selinux.el8.rpm: $(SRC_FILES) selinux-build.el8
	mkdir -p build
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		--net=host \
		--init \
		-v $(REPO_ROOT):/go/src/github.com/projectcalico/calico:rw \
		-w /go/src/$(PACKAGE_NAME) $(SELINUX_BUILD_IMAGE_EL8):latest \
		./build.sh

.PHONY: calico-selinux.el9.rpm
calico-selinux.el9.rpm: $(SRC_FILES) selinux-build.el9
	mkdir -p build
	docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		--net=host \
		--init \
		-v $(REPO_ROOT):/go/src/github.com/projectcalico/calico:rw \
		-w /go/src/$(PACKAGE_NAME) $(SELINUX_BUILD_IMAGE_EL9):latest \
		./build.sh

###############################################################################
# Clean
###############################################################################
clean:
	rm -f $(SELINUX_BUILD_CONTAINER_CREATED_EL8) $(SELINUX_BUILD_CONTAINER_CREATED_EL9)
	rm -f *.pp
	rm -fr build/
	rm -fr tmp/
	-docker image rm -f $$(docker images $(SELINUX_BUILD_IMAGE_EL8) -a -q)
	-docker image rm -f $$(docker images $(SELINUX_BUILD_IMAGE_EL9) -a -q)
