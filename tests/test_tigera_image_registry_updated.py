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

# create list of images for this release
with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_tigera_image_registry_updated():
    req = requests.get("%s/%s/getting-started/private-registry/private-registry-regular" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    print '[INFO] checking image registry update in docs uses {0} registry'.format(REGISTRY)

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Calico Enterprise images.").find_next('code')
    images = [x.replace('docker pull ', '') for x in page.text.split('\n') if re.search('tigera', x)]
    for image in images:
        assert re.search(REGISTRY, image)
        ver_image = image.replace('%s/tigera/' % REGISTRY, '').split(':')
        expected_ver = RELEASE_VERSION
        if ver_image[0] == 'operator':
          expected_ver = release.get('tigera-operator').get('version')
        if ver_image[0] == 'kibana':
          expected_ver = release['components'].get('kibana').get('version')
        if ver_image[0] == 'key-cert-provisioner':
          expected_ver = release['components'].get('key-cert-provisioner').get('version')
        print '[INFO] checking registry image %s references %s' % (ver_image[0], expected_ver)
        assert ver_image[1] == expected_ver

def test_non_tigera_image_registry_updated():
    req = requests.get("%s/%s/getting-started/private-registry/private-registry-regular" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    expected_images = {k: v for k, v in release.get('components').items() if v.has_key('image') and not v.get('image').startswith('tigera/')}

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Calico Enterprise images.").find_next('code')
    images = [x.replace('docker pull ', '') for x in page.text.split('\n') if not re.search('tigera', x) and not x == '']

    assert len(images) == len(expected_images)
    for image in images:
        print '[INFO] checking registry image %s' % (image)
        expected = {k:v for k, v in expected_images.items() if re.match(image, '{0}/{1}:{2}'.format(v.get('registry'), v.get('image'), v.get('version')))}
        assert len(expected) == 1
