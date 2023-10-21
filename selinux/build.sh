#!/bin/bash
set -e

make -f /usr/share/selinux/devel/Makefile calico.pp

cd build/

rpmbuild \
    --quiet \
    --define "_builddir $PWD" \
    --define "_buildrootdir $PWD/.build" \
    --define "_rpmdir $PWD/dist" \
    --define "_sourcedir $PWD/.." \
    --define "_specdir $PWD/.." \
    --define "_srcrpmdir $PWD/dist/source" \
    --define "_topdir $PWD/rpmbuild" \
    -ba ../calico-selinux.spec
