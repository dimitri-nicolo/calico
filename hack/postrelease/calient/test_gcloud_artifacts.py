#!/usr/bin/env python3

"""
Testing and validation for artifacts uploaded to Google
Cloud Storage
"""

import functools

import pytest

import variables
from variables import defaults
from lib.gcloud import GCBucket

pytestmark = [pytest.mark.gcloud]

@pytest.fixture(name="gstorage_validator")
@functools.cache
def get_gstorage_validator():
    """
    Return a (cached) version of the GCBucket
    object that we can use for validation
    """
    return GCBucket(
        bucket_name=defaults.TIGERA_GCLOUD_WINDOWS_BUCKET
    )

@pytest.mark.calico_release_cut
@pytest.mark.postrelease_artifacts
def test_windows_client_artifact_uploaded(gstorage_validator):
    gstorage_validator.get_windows_artifact_metadata(
        calico_version=defaults.CALICO_VERSION
    )