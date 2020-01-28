import os
import yaml
import requests
import tests


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
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION
    release_components = release.get('components')
    ALL_IMAGES = release_components.keys()
    

def test_release_tag_present():
    assert QUAY_API_TOKEN != 'fake-token', '[ERROR] need a real QUAY_API_TOKEN env value'

    version_mapped_images = {k: v for k, v in release_components.items() if v.has_key('image') and  not v.has_key('registry')}

    headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer %s' % QUAY_API_TOKEN}
    for _, v in version_mapped_images.items():
      repository = v.get('image')
      expected_ver = v.get('version')
      assert expected_ver == RELEASE_VERSION
      req_url = '{base_url}/repository/{repository}/tag/{tag}/images'.format(
          base_url=QUAY_API_URL, repository=repository, tag=expected_ver)
      print '[INFO] checking {0}:{1} at {2}'.format(repository, expected_ver, req_url)
      res = requests.get(req_url, headers=headers)
      assert res.status_code == 200
