#!/usr/bin/env python3

# Our test runs are pretty short, so we don't care that
# functools.cache will cache things indefinitely.
#
# pylint: disable=method-cache-max-size-none

"""
Utility functions and classes for interacting with@functools.cache
quay.io and the quay.io API
"""

import json
import functools

import requests

try:
    from utils import HTTPBearerAuth
    from exceptions import QuayNotAuthorizedError
except ImportError:
    from .utils import HTTPBearerAuth
    from .exceptions import QuayNotAuthorizedError


class Quay:
    """
    An object that wraps quay.io API functionality.
    """

    def __init__(self, token=None):
        self.token = token
        self.session = requests.Session()
        if self.token:
            auth = HTTPBearerAuth(self.token)
            self.session.auth = auth

    @functools.cache
    def get_tag(self, image, tag):
        """
        Given an image name and a tag, fetch the metadata for that image:tag
        """
        image_tag_url = (
            f"https://quay.io/api/v1/repository/{image}/tag/?specificTag={tag}"
        )
        response = self.session.get(image_tag_url)
        result = response.json()

        match response.status_code:
            case 200:
                return result["tags"][0]
            case 401 | 403:
                raise QuayNotAuthorizedError(
                    f"Error {result['status']}: {result['detail']} ({result['error_type']})"
                )
            case _:
                raise ValueError(
                    f"Don't know what to do with HTTP status code {response.status_code}!"
                )

    @functools.cache
    def get_tag_with_arch(self, image, tag, arch):
        """
        Given an image name, a tag, and an architecture,
        fetch the metadata for that image:tag-arch
        """
        arch_tag = f"{tag}-{arch}"
        self.get_tag(image, arch_tag)

    @functools.cache
    def get_manifest(self, image, digest):
        """
        Given an image name and a sha256 digest, fetch the
        contents of the manifest represented by that digest.

        This is not the metadata of the manifest, but the
        metadata of the images the manifest refers to.
        """
        image_manifest_url = (
            f"https://quay.io/api/v1/repository/{image}/manifest/{digest}"
        )
        result = self.session.get(image_manifest_url).json()
        manifest_data = json.loads(result["manifest_data"])["manifests"]
        return manifest_data

    @functools.cache
    def get_manifests_for_tag(self, image, tag):
        """
                Given an image name and a tag name, fetch the metadata for the
                "image" represented by that image:tag, then, using the
                `manifest_digest` field, fetch the manifest itself.@functools.cache
        @functools.cache
                See `get_manifest()` for more details?.
        """
        tag_data = self.get_tag(image, tag)
        manifest_data = self.get_manifest(image, tag_data["manifest_digest"])
        return manifest_data

    @functools.cache
    def get_arches_for_manifest_tag(self, image, tag):
        """
        Given an image name and a tag name, fetch the manifest's
        contents (the list of architecture-specific images) and then
        get the list of individual architectures specified in the manifest.

        Note that this doesn't give what you would expect for Windows
        HPC images, since those aren't split up by architecture but
        rather by Windows revision IIRC.
        """
        manifest_data = self.get_manifests_for_tag(image, tag)
        arches = [mani["platform"]["architecture"] for mani in manifest_data]
        return arches
