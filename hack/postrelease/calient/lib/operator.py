#!/usr/bin/env python3

# Our test runs are pretty short, so we don't care that
# functools.cache will cache things indefinitely.
#
# pylint: disable=method-cache-max-size-none

"""
Generic utility functions and classes for testing Operator
"""

import json
import functools
import subprocess

from .utils import release_url, tag_url, http_get_is_200
from .log_config import log

OPERATOR_SLUG = "tigera/operator"
OPERATOR_REGISTRY = "quay.io"
OPERATOR_MANIFEST_CMD = ["docker", "manifest", "inspect"]


class Operator:
    """
    An object to represent an Operator version (potentially including
    a specific architecture), metadata about that operator, data
    acquired from the operator image, and more.'
    """

    def __init__(  # pylint: disable=too-many-arguments
        self,
        version,
        project=OPERATOR_SLUG,
        registry=OPERATOR_REGISTRY,
        arch=None,
        root=None,
    ):
        self.project = project
        self.version = version
        self.registry = registry
        self.arch = arch
        self.__manifest = None
        self.root = root

    @property
    def release_url(self):
        """
        Get the URL to the Github release for this version of the operator.
        """
        return release_url(self.project, self.version)

    @property
    def tag_url(self):
        """
        Get the URL to the Github tag for this version of the operator.
        """
        return tag_url(self.project, self.version)

    @property
    def image_name(self):
        """
        Get the URL (with registry) to the docker image for this version of the operator.
        """
        return self.project

    @property
    def image_tag(self):
        """
        Get the URL (with registry and tag name) to the docker image for this
        version of the operator.
        """
        return f"{self.image_name}:{self.version}{'-' + self.arch if self.arch else ''}"

    @property
    def tag(self):
        """
        Get the tag-portion (i.e. the part of the image name after the :) for this
        version of the operator, including the architecture if this Operator object
        represents one.
        """
        return f"{self.version}{'-' + self.arch if self.arch else ''}"

    def with_arch(self, arch):
        """
        Given the current Operator object, return an Operator object that represents
        the same version but a specific architecture.
        """
        return Operator(self.version, self.project, self.registry, arch, root=self)

    def with_all_arches(self):
        """
        Given the current Operator object, get a list of all valid architectures for
        this manifest and return an Operator object for each architecture.
        """
        return [self.with_arch(arch) for arch in self.arches]

    @functools.cache
    def get_manifest(self):
        """
        Fetch (and cache) the manifest from Docker for the current Operator object
        using `docker manifest inspect`.
        """
        if self.arch:
            raise ValueError(
                "Manifests should be fetched from a root Operator object "
                "(you can  try `obj.root.manifest` instead of `obj.manifest`)"
            )

        cmd = OPERATOR_MANIFEST_CMD + [self.image_tag]
        with subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE
        ) as proc:
            ret = proc.wait()
            # ret == 0 if our process returned successfully; parse the data we got
            # back and then return it
            if ret == 0:
                output = proc.stdout.read()
                manifest_data = json.loads(output)
                self.__manifest = manifest_data
                return self.__manifest
            # ...otherwise, we got an error. Get the error message from stderr...
            err_msg = proc.stderr.read().decode()
            # ...and if it says there's "no such manifest", raise an error for that;
            if "no such manifest" in err_msg:
                raise RuntimeError(
                    f"Specified manifest {self.image_tag} does not exist"
                )
            # otherwise, raise a more generic message and include the error message from stderr
            # itself.
            raise RuntimeError(f"Unhandled error occurred fetching manifest: {err_msg}")

    @functools.cache
    def get_image_list(self):
        """
        Fetches the list of Calico images from the actual operator image. Note
        that this is not an exhaustive list of images! For example, it does not
        include the operator image itself (reasonable).
        """
        log.info("Fetching image list from operator %s", self.image_tag)
        images_cmd = [
            "docker",
            "run",
            "--quiet",
            "--rm",
            self.image_tag,
            "--print-images=list",
        ]
        with subprocess.Popen(
            images_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE
        ) as proc:
            ret = proc.wait()
            if ret == 0:
                output = proc.stdout.read().decode().splitlines()
                return output
            raise RuntimeError("Fetching images list from operator image failed")

    @property
    @functools.cache
    def manifest(self):
        """
        Call `get_manifest()` to ensure that we have the manifest data cached locally,
        and then return it.
        """
        self.get_manifest()
        return self.__manifest

    @property
    def arches(self):
        """
        Return the list of architectures fetched from the Operator object's manifest.
        """
        return [mani["platform"]["architecture"] for mani in self.manifest["manifests"]]

    def tag_exists(self):
        """
        Returns true if fetching `self.tag_url` returns HTTP 200
        """
        return http_get_is_200(self.tag_url)

    def release_exists(self):
        """
        Returns true if fetching `self.release_url` returns HTTP 200
        """
        return http_get_is_200(self.release_url)

    def __repr__(self):
        params = [
            f"Operator '{self.project}'",
            f"version '{self.version}'",
        ]
        if self.arch:
            params.append(f"arch '{self.arch}'")
        return f"<{', '.join(params)}>"
