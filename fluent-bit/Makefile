include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/fluent-bit

FLUENT_BIT_VERSION = 3.1.4

##############################################################################
# Download and include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

###############################################################################
# Build
###############################################################################

clean:
	rm -f ${FLUENT_BIT_DOWNLOADED}
	rm -f *.tar.gz
	rm -fr package

###############################################################################
# Package
###############################################################################
FLUENT_BIT_DOWNLOADED=.fluent-bit.downloaded

.PHONY: init-source
# This target downloads fluent-bit source archive, extract it and rename
# fluent-bit folder name to calico-fluent-bit for better packaging.
init-source: $(FLUENT_BIT_DOWNLOADED)
$(FLUENT_BIT_DOWNLOADED):
	mkdir -p calico-fluent-bit-${FLUENT_BIT_VERSION}
	curl -sfL https://github.com/fluent/fluent-bit/archive/refs/tags/v${FLUENT_BIT_VERSION}.tar.gz | tar xz --strip-components 1 -C calico-fluent-bit-${FLUENT_BIT_VERSION}
	tar czf v${FLUENT_BIT_VERSION}.tar.gz calico-fluent-bit-${FLUENT_BIT_VERSION}
	rm -fr calico-fluent-bit-${FLUENT_BIT_VERSION}
	touch $@

.PHONY: package
package: calico-fluent-bit.rhel8 calico-fluent-bit.rhel9

.PHONY: calico-fluent-bit.rhel8
calico-fluent-bit.rhel8: ${FLUENT_BIT_DOWNLOADED} host-native-build rhel/calico-fluent-bit.spec
	$(call host_native_rpm_build,rhel8,calico-fluent-bit)

.PHONY: calico-fluent-bit.rhel9
calico-fluent-bit.rhel9: ${FLUENT_BIT_DOWNLOADED} host-native-build rhel/calico-fluent-bit.spec
	$(call host_native_rpm_build,rhel9,calico-fluent-bit)

###############################################################################
# CI
###############################################################################
.PHONY: ci
ci: clean package
