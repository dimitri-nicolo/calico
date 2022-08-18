import os
import re
import requests
import yaml
import tests
from bs4 import BeautifulSoup
from subprocess import check_output

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
DOCS_URL = tests.DOCS_URL
GIT_HASH = tests.GIT_HASH

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[0].get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION


def test_updated_docs_deployed():
    req = requests.get('%s/%s/release-notes' % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    git_hash = BeautifulSoup(req.content, features="html.parser").find("div", {"class":"git-hash"})
    assert GIT_HASH == git_hash.attrs['id'], "%s did not match %s" % (GIT_HASH, git_hash.attrs['id'])


def test_latest_redirects_correctly():
    req = requests.get("%s/latest" % DOCS_URL)
    assert req.status_code == 200


def test_latest_release_notes_redirects_correctly():
    req = requests.get("%s/latest/release-notes" % DOCS_URL)
    assert req.status_code == 200


def test_release_version_redirects_correctly():
    req = requests.get("%s/%s/" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    req = requests.get("%s/%s/release-notes/" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

def test_private_registry_docs_match_operator_images():
    req = requests.get("%s/%s/getting-started/private-registry/private-registry-regular" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    print('[INFO] checking image version match those operator uses')

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Calico Enterprise images.").find_next('code')
    images = [x.replace('docker pull ', '') for x in page.text.split('\n')]
    
    operator_image = [x for x in images if re.search('tigera/operator', x)][0]
    output = check_output('docker run {} --print-images=list'.format(operator_image), shell=True).decode('utf-8')
    operator_images = [x for x in output.split('\n')]
    
    for image in [x for x in images if x and not re.search('tigera/operator', x)]:
      img_deets = image.split(':')
      img_name = img_deets[0]
      match = [x for x in operator_images if re.search(img_name, x)]
      if len(match) > 0:
          img_version = img_deets[1]
          op_version = match[0].split(':')[1]
          assert img_version == op_version, '{0} in doc ({1}) does not match operator version({2})'.format(img_name, img_version, op_version)
