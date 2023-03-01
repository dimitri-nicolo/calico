#!/bin/bash
set -e -x

make -f /usr/share/selinux/devel/Makefile calico-enterprise.pp

cd build/

rpmbuild \
    --define "_builddir $PWD" \
    --define "_buildrootdir $PWD/.build" \
    --define "_rpmdir $PWD/dist" \
    --define "_sourcedir $PWD/.." \
    --define "_specdir $PWD/.." \
    --define "_srcrpmdir $PWD/dist/source" \
    --define "_topdir $PWD/rpmbuild" \
    -ba ../calico-enterprise-selinux.spec
