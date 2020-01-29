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
    

@parameterized(VERSION_MAPPED_IMAGES.items())
def test_release_tag_present(name, component):
    assert QUAY_API_TOKEN != 'fake-token', '[ERROR] need a real QUAY_API_TOKEN env value'
    
    print '[INFO] checking quay image posted for {0} with {1} tag'.format(name, RELEASE_VERSION)
    
    headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer {}'.format(QUAY_API_TOKEN)}
    repository = component.get('image')
    expected_ver = component.get('version')
    assert expected_ver == RELEASE_VERSION
    req_url = '{base_url}/repository/{repository}/tag/{tag}/images'.format(
        base_url=QUAY_API_URL, repository=repository, tag=expected_ver)
    res = requests.get(req_url, headers=headers)
    assert res.status_code == 200
