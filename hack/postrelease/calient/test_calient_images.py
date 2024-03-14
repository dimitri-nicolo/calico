#!/usr/bin/env python3

"""
Tests for ensuring calico images exist and are pushed.
"""

import yaml
import pytest

from variables import defaults

from lib.quay import Quay

def versions_data():
    with open(defaults.VERSIONS_YAML_FILE, encoding="utf8") as input_versions_file:
        versions_data = yaml.safe_load(input_versions_file)[0]
    return versions_data

def operator_images():
    """
    Take the parsed version data from versions.yaml
    and create a list of images (including the operator
    image) and return that.
    """
    yaml_data = versions_data()
    components = yaml_data["components"]
    operator = yaml_data["tigera-operator"]
    components["tigera-operator"] = operator

    images_list = []
    for component in components.values():
        if "image" in component:
            images_list.append((component["image"], component["version"]))
    return sorted(images_list)


@pytest.mark.quay
@pytest.mark.calico_release_cut
@pytest.mark.parametrize("calico_image,calico_tag", operator_images())
def test_calico_images(calico_image, calico_tag):
    """
    For the given image (fetched from versions.yaml), fetch
    it from Quay.io (via the API) to ensure it exists.
    """
    if calico_image in defaults.CALICO_IMAGES_SKIP.split():
        pytest.skip(f"Image {calico_image} found in skiplist.")
    quay = Quay(token=defaults.QUAY_TOKEN)
    quay.get_tag(calico_image, calico_tag)
