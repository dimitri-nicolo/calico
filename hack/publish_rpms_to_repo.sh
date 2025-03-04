#!/bin/bash

set -eu

# Just a note on odd syntax: putting >&2 in front of a line redirects
# stdout to stderr, which is a convenient shorthand to print error messages
# in POSIX shells.

if [[ ! -v VERSION ]]; then
    >&2 echo "This script requires VERSION to be set"
    exit 1
fi

# Check to see if we currently have any RPM files that were built
# for master (i.e. have a version of 0.0.0). If so, we don't want
# to publish them. Force the user to publish manually or clean
# first.
master_rpm_files=$(find selinux node fluent-bit -name '*0.0.0*')

if [[ -n ${master_rpm_files} ]]; then
    >&2 echo "The following files were built with an internal version number and cannot be published using this script:"
    echo "$master_rpm_files" | sed -e 's/^/    /'
    exit 1
fi

# Use sed to extract v<major>.<minor> from the version string
version=$(echo "${VERSION}" | sed -E 's/(v[0-9]\.[0-9]+).*/\1/')
tmpdir=$(mktemp --directory --tmpdir "non-cluster-host-build-${version}.XXXXXXXXX")
destpath=${tmpdir}/${version}

echo "Temporary directory is ${destpath}"

# The target path we store our repository at
remote_repo_path=s3://tigera-public/ee/rpms/${version}/

echo "Downloading repo metadata AWS S3"
# Download the remote repository except for the RPM files
aws --profile helm s3 cp --recursive --exclude '*.rpm' --quiet "${remote_repo_path}" "${destpath}"

echo "Copying new RPM files into temporary directory"
# For each of the packages that builds RPMs, rsync them into our temporary directory
for source in selinux node fluent-bit; do
    echo "    Copying files for ${source}"
    rsync --recursive --prune-empty-dirs --exclude="BUILD" ${source}/package/ "${destpath}" | ts "[${source}]"
done

echo "Generating updated repository metadata for new RPM files"
# For each of the two RHEL releases
for release in rhel8 rhel9; do
    (
        # Go into the release directory
        cd "${destpath}/${release}"
        # Make a list of all of the RPM files relative to the release directoryr
        find . -name '*.rpm' | sed -e 's/^\.\///' > filelist.txt
        createrepo_c \
            --update \
            --recycle-pkglist \
            --skip-stat \
            --pkglist=filelist.txt \
            --xz \
            --baseurl="https://downloads.tigera.io/ee/rpms/${version}/${release}/" \
            . | ts "[${release}]"
        rm filelist.txt
    )
done

echo "Uploading new RPMs and updated metadata to S3"
aws --profile helm s3 sync "${destpath}" "${remote_repo_path}" --acl public-read
