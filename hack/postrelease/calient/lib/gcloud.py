#!/usr/bin/env python3

try:
    from . import exceptions
except ImportError:
    import exceptions

from google.cloud import storage

class GCBucket:
    """
    Represents a Google Cloud storage bucket and
    provides methods to access predefined objects
    within (mostly to save on string formatting
    and concatenation in the tests themselves)
    """

    def __init__(self, bucket_name):
        # Google Cloud sessions and clients
        self.bucket_name = bucket_name
        self.storage_client = storage.Client()

        self.bucket = self.storage_client.get_bucket(self.bucket_name)

    def __get_object(self, key):
        """
        Given an object's key, fetch the metadata of the object
        from GC Storage and return it, or raise GCObjectNotFoundError
        if the key does not exist.
        """
        object_metadata = self.bucket.get_blob(key)
        if object_metadata is None:
            raise exceptions.GoogleStorageObjectNotFoundError(self.bucket_name, key)
        return object_metadata

    def get_windows_artifact_metadata(self, calico_version):
        object_key = f"tigera-calico-windows-{calico_version}.zip"
        return self.__get_object(object_key)
