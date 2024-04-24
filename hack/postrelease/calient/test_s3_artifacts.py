#!/usr/bin/env python3

"""
Testing and validation for artifacts uploaded to S3

TODO: also cloudfront
"""

import os
import sys
import pathlib
import functools

import xdist
import pytest

import variables
from variables import defaults
from lib.aws import S3Bucket
from lib.utils import get_version_from_specfile

pytestmark = [pytest.mark.aws, pytest.mark.filterwarnings("ignore:.*Use timezone-aware objects to represent datetimes in UTC.*:DeprecationWarning")]

@pytest.fixture(name="s3_validator")
@functools.cache
def get_s3_validator():
    """
    Return a (cached) version of the S3Bucket
    object that we can use for validation
    """
    return S3Bucket(
        bucket_name=defaults.TIGERA_S3_PUBLIC_BUCKET, profile_name=defaults.AWS_PROFILE
    )


@pytest.fixture(name="local_manifests")
def list_local_manifests():
    """
    Fetch and return the list of manifests which currently exist locally.
    """
    manifests_dir = pathlib.Path(defaults.MANIFESTS_PATH)
    manifest_list = list(
        manifest.relative_to(defaults.MANIFESTS_PATH).as_posix()
        for manifest in manifests_dir.rglob("*")
        if manifest.is_file()
    )
    return manifest_list


@pytest.mark.postrelease_artifacts
def test_manifests_uploaded(s3_validator, local_manifests):
    """
    Validate that the list of manifesets we get from S3 matches
    the list of manifests we have locally in the repository.

    Note: this is going to break when we add or remove manifests.

    TODO: get the list of manifests from git rather than the local filesystem?
    """
    s3_manifests = s3_validator.get_manifests_list(
        calico_version=defaults.CALICO_VERSION
    )

    for manifest_skip_file in variables.AWS_MANIFESTS_SKIP_FILES:
        if manifest_skip_file in s3_manifests:
            s3_manifests.remove(manifest_skip_file)
        if manifest_skip_file in local_manifests:
            local_manifests.remove(manifest_skip_file)

    assert sorted(s3_manifests) == sorted(local_manifests)


@pytest.mark.postrelease_artifacts
def test_helm_chart_uploaded(s3_validator):
    """
    Make sure that the helm chart has been uploaded to the
    correct location
    """
    s3_validator.get_chart_metadata(
        calico_version=defaults.CALICO_VERSION, chart_release=defaults.CHART_RELEASE
    )


@pytest.mark.postrelease_artifacts
def test_release_archive_uploaded(s3_validator):
    """
    Make sure that the release archive has been uploaded to
    the correct location
    """
    s3_validator.get_release_archive_metadata(
        calico_version=defaults.CALICO_VERSION,
        operator_version=defaults.OPERATOR_VERSION,
    )


@pytest.mark.postrelease_artifacts
@pytest.mark.parametrize("rhel_version", variables.SELINUX_RHEL_VERSIONS)
def test_selinux_rpms_uploaded(s3_validator, rhel_version):
    """
    Validate that the selinux RPMs have been uploaded to S3 for the given
    supported RHEL version
    """
    selinux_rpm_version = get_version_from_specfile(defaults.SELINUX_SPECFILE_PATH)
    file_key = (
        f"ee/archives/calico-selinux-{selinux_rpm_version}.{rhel_version}.noarch.rpm"
    )
    s3_validator.get_file_metadata(file_key)

@pytest.mark.calico_release_cut
@pytest.mark.parametrize("binary_name", variables.CALIENT_UPLOADED_BINARIES)
def test_client_binaries_uploaded(s3_validator, binary_name):
    """
    Validate that the client binaries were uploaded to S3
    """
    file_key = f"ee/binaries/{defaults.CALICO_VERSION}/{binary_name}"
    s3_validator.get_file_metadata(file_key)
