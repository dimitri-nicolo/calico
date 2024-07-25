#!/bin/bash
set -e

cd "build/$1"

rpmbuild \
    --quiet \
    --define "_builddir $PWD" \
    --define "_buildrootdir $PWD/.build" \
    --define "_rpmdir $PWD/dist" \
    --define "_sourcedir $PWD/../.." \
    --define "_specdir $PWD/../.." \
    --define "_srcrpmdir $PWD/dist/source" \
    --define "_topdir $PWD/rpmbuild" \
    -ba ../../calico-selinux.spec
