#!/usr/bin/env python3

"""
Functionality and classes for interacting with AWS resources,
such as S3 buckets and the objects they contain.

"""

import pathlib
import functools

import boto3

from botocore import credentials
import botocore.session
import botocore.errorfactory

try:
    from . import exceptions
except ImportError:
    import exceptions


@functools.cache
def bc_credential_cache_session():
    """
    Set up a botocore session object and configure it to use a local
    credential cache (e.g. the one used by the AWS CLI tool).

    This session cache can then be passed to a boto3.Session
    initializer to prevent us from having to invoke 2FA for every
    S3-related test.
    """
    cli_cache = pathlib.Path("~/.aws/cli/cache").expanduser().as_posix()
    session = botocore.session.get_session()
    session.set_config_variable("profile", "helm")
    session.get_component("credential_provider").get_provider(
        "assume-role"
    ).cache = credentials.JSONFileCache(cli_cache)
    return session


class S3Bucket:
    """
    Represents an S3 bucket and provides methods
    to access predefined objects within (mostly
    to save on string formatting and concatenation
    in the tests themselves)
    """

    def __init__(self, bucket_name, profile_name):
        # AWS sessions and clients
        self.profile_name = profile_name
        self.session = boto3.Session(
            profile_name=self.profile_name,
            botocore_session=bc_credential_cache_session(),
        )
        self.s3_resource = self.session.resource("s3")
        self.s3_client = self.session.client("s3")
        # Remember our bucket name, and get a reference to the bucket itself
        self.bucket_name = bucket_name
        self.bucket = self.s3_resource.Bucket(self.bucket_name)

    def __get_object(self, key):
        """
        Given an object's key, fetch the metadata of the object
        from S3 and return it, or raise S3ObjectNotFoundError
        if the key does not exist.
        """
        try:
            object_metadata = self.s3_client.get_object(
                Bucket=self.bucket_name, Key=key
            )
            return object_metadata
        except self.s3_client.exceptions.NoSuchKey as exc:
            raise exceptions.S3ObjectNotFoundError(self.bucket_name, key) from exc

    def get_manifests_list(self, calico_version):
        """
        Get all the manifests from S3 which have the
        given version prefix. We can then validate this
        against the list of manifests we expect to have
        uploaded.
        """
        key_prefix = f"ee/{calico_version}/manifests/"
        objects_gen = self.bucket.objects.filter(Prefix=key_prefix)
        existing_files = sorted(
            [bucket_obj.key.replace(key_prefix, "") for bucket_obj in objects_gen]
        )
        return existing_files

    def get_chart_metadata(self, calico_version, chart_release):
        """
        Given the calico version and chart release, fetch the metadata
        for the Tigera operator chart (or raise S3ObjectNotFoundError if not present)
        """
        object_key = f"ee/charts/tigera-operator-{calico_version}-{chart_release}.tgz"
        return self.__get_object(object_key)

    def get_release_archive_metadata(self, calico_version, operator_version):
        """
        Given a calico and operator version, fetch the metadata for the
        release archive (or raise S3ObjectNotFoundError if not present)

        """
        object_key = f"ee/archives/release-{calico_version}-{operator_version}.tgz"
        return self.__get_object(object_key)

    def get_file_metadata(self, object_key):
        """
        Given an arbitrary object key (i.e. file path), fetch the metadata
        for the given object (or raise S3ObjectNotFoundError if not present)
        """
        return self.__get_object(object_key)
