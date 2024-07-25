include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/selinux

##############################################################################
# Download and include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Build RPM
###############################################################################
AWS_PROFILE ?= helm

SRC_FILES = calico-selinux.spec \
	calico.fc \
	calico.if \
	calico.te

.PHONY: build
build: calico-selinux.rhel8 calico-selinux.rhel9

calico.pp: $(SRC_FILES) host-native-build
	$(DOCKER_HOST_NATIVE_RUN) $(HOST_NATIVE_BUILD_IMAGE):rhel8 \
		sh -c 'make -f /usr/share/selinux/devel/Makefile calico.pp'

.PHONY: calico-selinux.rhel8
calico-selinux.rhel8: calico.pp
	$(info *** Building SELinux RPM for RHEL 8)
	@mkdir -p build/rhel8
	$(DOCKER_HOST_NATIVE_RUN) $(HOST_NATIVE_BUILD_IMAGE):rhel8 ./build.sh rhel8

.PHONY: calico-selinux.rhel9
calico-selinux.rhel9: calico.pp
	$(info *** Building SELinux RPM for RHEL 9)
	@mkdir -p build/rhel9
	$(DOCKER_HOST_NATIVE_RUN) $(HOST_NATIVE_BUILD_IMAGE):rhel9 ./build.sh rhel9

publish: publish-rhel8 publish-rhel9

publish-rhel8: calico-selinux.rhel8
	$(info *** Publishing SELinux RPM for RHEL 8)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard build/rhel8/dist/noarch/calico-selinux-*.el8.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

publish-rhel9: calico-selinux.rhel9
	$(info *** Publishing SELinux RPM for RHEL 9)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard build/rhel9/dist/noarch/calico-selinux-*.el9.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

###############################################################################
# Clean
###############################################################################
.PHONY: clean
clean:
	rm -f *.pp
	rm -fr build/
	rm -fr tmp/

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
