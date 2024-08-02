include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/selinux

CALICO_SELINUX_VERSION = 1.0

##############################################################################
# Download and include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Package
###############################################################################
AWS_PROFILE ?= helm

SRC_FILES = rhel/calico-selinux.spec.in \
	calico.fc \
	calico.if \
	calico.te

.PHONY: package
package: calico-selinux.rhel8 calico-selinux.rhel9

calico.pp: $(SRC_FILES) host-native-build
	$(DOCKER_HOST_NATIVE_RUN) $(HOST_NATIVE_BUILD_IMAGE):rhel8 \
		sh -c 'make -f /usr/share/selinux/devel/Makefile calico.pp'

.PHONY: calico-selinux.rhel8
calico-selinux.rhel8: calico.pp
	$(info *** Building SELinux RPM for RHEL 8)
	sed 's/@CALICO_SELINUX_VERSION@/$(CALICO_SELINUX_VERSION)/g' rhel/calico-selinux.spec.in > rhel/calico-selinux.spec
	$(call host_native_rpm_build,rhel8,calico-selinux)

.PHONY: calico-selinux.rhel9
calico-selinux.rhel9: calico.pp
	$(info *** Building SELinux RPM for RHEL 9)
	sed 's/@CALICO_SELINUX_VERSION@/$(CALICO_SELINUX_VERSION)/g' rhel/calico-selinux.spec.in > rhel/calico-selinux.spec
	$(call host_native_rpm_build,rhel9,calico-selinux)

publish: publish-rhel8 publish-rhel9

publish-rhel8: calico-selinux.rhel8
	$(info *** Publishing SELinux RPM for RHEL 8)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard package/rhel8/RPMS/noarch/calico-selinux-*.el8.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

publish-rhel9: calico-selinux.rhel9
	$(info *** Publishing SELinux RPM for RHEL 9)
	@aws --profile $(AWS_PROFILE) s3 cp $(wildcard package/rhel9/RPMS/noarch/calico-selinux-*.el9.*.rpm) s3://tigera-public/ee/archives/ --acl public-read

###############################################################################
# Clean
###############################################################################
.PHONY: clean
clean:
	rm -f *.pp
	rm -f rhel/calico-selinux.spec
	rm -fr package/
	rm -fr tmp/

###############################################################################
# Tests
###############################################################################
ZONE ?= us-central1-a
VM_PREFIX ?= $(USER)-selinux

.PHONY: fv
fv: package
	fv/create-vm-and-install.sh $(ZONE) $(VM_PREFIX) $(CALICO_SELINUX_VERSION)
	fv/run-tests.sh $(ZONE) $(VM_PREFIX)

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean fv
