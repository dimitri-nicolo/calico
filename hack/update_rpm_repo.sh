#!/bin/bash

set -eu

version=${VERSION}
tmpdir=$(mktemp --directory --tmpdir non-cluster-host-build-${version}.XXXXXXXXX)
destpath=${tmpdir}/${version}

echo "Temporary directory is ${destpath}"

# The target path we store our repository at
remote_repo_path=s3://tigera-public/ee/rpms/${version}/

echo "Downloading repo metadata AWS S3"
# Download the remote repository except for the RPM files
aws --profile helm s3 cp --recursive --exclude '*.rpm' --quiet ${remote_repo_path} ${destpath}

echo "Copying new RPM files into temporary directory"
# For each of the packages that builds RPMs, rsync them into our temporary directory
for source in selinux node fluent-bit; do
    echo "    Copying files for ${source}"
    rsync --recursive --prune-empty-dirs --exclude="BUILD" ${source}/package/ ${destpath} | ts "[${source}]"
done

echo "Generating updated repository metadata for new RPM files"
# For each of the two RHEL releases
for release in rhel8 rhel9; do
    (
        # Go into the release directory
        cd ${destpath}/${release}
        # Make a list of all of the RPM files relative to the release directoryr
        find . -name '*.rpm' | sed -e 's/^\.\///' > filelist.txt
        createrepo_c \
            --update \
            --recycle-pkglist \
            --skip-stat \
            --pkglist=filelist.txt \
            --xz \
            --baseurl=https://downloads.tigera.io/ee/rpms/${version}/${release}/ \
            . | ts "[${release}]"
        rm filelist.txt
    )
done

echo "Uploading new RPMs and updated metadata to S3"
aws --profile helm s3 sync ${destpath} ${remote_repo_path} --acl public-read
