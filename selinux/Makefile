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

AWS_PROFILE ?= helm

.PHONY: selinux-build.el8 selinux-build.el9

docker-containers: selinux-build.el8 selinux-build.el9

selinux-build.el8: $(SELINUX_BUILD_IMAGE_EL8) clean-files
$(SELINUX_BUILD_IMAGE_EL8): $(SELINUX_BUILD_CONTAINER_CREATED_EL8)
$(SELINUX_BUILD_CONTAINER_CREATED_EL8): Dockerfile.el8
	$(DOCKER_BUILD) -t $(SELINUX_BUILD_IMAGE_EL8):latest -f Dockerfile.el8 .
	touch $@

selinux-build.el9: $(SELINUX_BUILD_IMAGE_EL9) clean-files
$(SELINUX_BUILD_IMAGE_EL9): $(SELINUX_BUILD_CONTAINER_CREATED_EL9)
$(SELINUX_BUILD_CONTAINER_CREATED_EL9): Dockerfile.el9
	$(DOCKER_BUILD) -t $(SELINUX_BUILD_IMAGE_EL9):latest -f Dockerfile.el9 .
	touch $@

###############################################################################
# Build RPM
###############################################################################
.PHONY: build
build: calico-selinux.el8.rpm calico-selinux.el9.rpm

.PHONY: calico-selinux.el8.rpm
calico-selinux.el8.rpm: $(SRC_FILES) selinux-build.el8
	$(info *** Building selinux RPM for el8)
	@mkdir -p build
	@docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		--net=host \
		--init \
		-v $(REPO_ROOT):/go/src/github.com/projectcalico/calico:rw \
		-w /go/src/$(PACKAGE_NAME) $(SELINUX_BUILD_IMAGE_EL8):latest \
		./build.sh

.PHONY: calico-selinux.el9.rpm
calico-selinux.el9.rpm: $(SRC_FILES) selinux-build.el9
	$(info *** Building selinux RPM for el9)
	@mkdir -p build
	@docker run --rm --user $(LOCAL_USER_ID):$(LOCAL_GROUP_ID) \
		--net=host \
		--init \
		-v $(REPO_ROOT):/go/src/github.com/projectcalico/calico:rw \
		-w /go/src/$(PACKAGE_NAME) $(SELINUX_BUILD_IMAGE_EL9):latest \
		./build.sh

publish: publish-el8 publish-el9

publish-el8: calico-selinux.el8.rpm | calico-selinux.el9.rpm
	$(info *** Publishing selinux RPM for el8)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard build/dist/noarch/calico-selinux-*.el8.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

publish-el9: calico-selinux.el9.rpm | calico-selinux.el8.rpm
	$(info *** Publishing selinux RPM for el9)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard build/dist/noarch/calico-selinux-*.el9.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

###############################################################################
# Clean
###############################################################################
.PHONY: clean
clean: clean-files clean-docker

clean-files: 
	rm -f *.pp
	rm -fr build/
	rm -fr tmp/
clean-docker:
	rm -f $(SELINUX_BUILD_CONTAINER_CREATED_EL8) $(SELINUX_BUILD_CONTAINER_CREATED_EL9)
	-docker image rm -f $$(docker images $(SELINUX_BUILD_IMAGE_EL8) -a -q)
	-docker image rm -f $$(docker images $(SELINUX_BUILD_IMAGE_EL9) -a -q)

###############################################################################
# Tests
###############################################################################
ZONE ?= us-central1-a
VM_PREFIX ?= $(USER)-selinux

.PHONY: fv
fv: build
	fv/create-vm-and-install.sh $(ZONE) $(VM_PREFIX)
	fv/run-tests.sh $(ZONE) $(VM_PREFIX)

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean fv
