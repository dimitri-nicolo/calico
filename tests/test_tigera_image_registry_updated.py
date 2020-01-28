import os
import yaml
import re
import requests
import tests
from bs4 import BeautifulSoup

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
DOCS_URL = tests.DOCS_URL
REGISTRY = tests.REGISTRY

EXCLUDED_IMAGES = [
  'operator',
  'kibana',
]

# create list of images for this release
with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[0].get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_image_registry_updated():
    req = requests.get("%s/%s/getting-started/private-registry" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Calico Enterprise images.").find_next('code')
    images = [x.replace('docker pull ', '') for x in page.text.split('\n') if re.search('tigera', x)]
    for image in images:
        assert re.search(REGISTRY, image)
        ver_image = image.replace('%s/tigera/' % REGISTRY, '').split(':')
        if ver_image[0] in EXCLUDED_IMAGES:
          continue
        else:
          print '[INFO] checking registry image %s references %s version' % (ver_image[0], RELEASE_VERSION)
          assert ver_image[1] == RELEASE_VERSION
