import os
import yaml
import requests
import tests
from parameterized import parameterized


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
QUAY_REGISTRY = tests.QUAY_REGISTRY
QUAY_API_URL = tests.QUAY_API_URL
QUAY_API_TOKEN = tests.QUAY_API_TOKEN

# create list of images for this release
with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: {0}'.format(RELEASE_VERSION)
    VERSION_MAPPED_IMAGES = {k: v for k, v in release.get('components').items() if v.has_key('image') and v.get('image').startswith('tigera/')}
    # add tigera-operator to VERSION_MAPPED_IMAGES to check tigera-operator release tag
    VERSION_MAPPED_IMAGES.update({'tigera-operator':release.get('tigera-operator')})
    headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer {}'.format(QUAY_API_TOKEN)}


@parameterized(VERSION_MAPPED_IMAGES.items())
def test_release_tag_present(name, component):
    assert QUAY_API_TOKEN != 'fake-token', '[ERROR] need a real QUAY_API_TOKEN env value'

    image_name = component.get('image')
    expected_ver = component.get('version')
    print '[INFO] checking quay image posted for {0} with {1} tag'.format(name, expected_ver)
    req_url = '{base_url}/repository/{image_name}/tag/{tag}/images'.format(
        base_url=QUAY_API_URL, image_name=image_name, tag=expected_ver)
    res = requests.get(req_url, headers=headers)
    assert res.status_code == 200
