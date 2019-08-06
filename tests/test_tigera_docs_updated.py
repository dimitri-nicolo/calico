import os
import requests
import yaml
import tests
from bs4 import BeautifulSoup

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
DOCS_URL = tests.DOCS_URL
GIT_HASH = tests.GIT_HASH

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION


def test_updated_docs_deployed():
    req = requests.get("%s/master/release-notes" % DOCS_URL)
    assert req.status_code == 200

    git_hash = BeautifulSoup(req.content, features="html.parser").find("div", {"class":"git-hash"})
    assert GIT_HASH == git_hash.attrs['id'], "%s did not match %s" % (GIT_HASH, git_hash.attrs['id'])


def test_latest_redirects_correctly():
    req = requests.get("%s/latest" % DOCS_URL)
    assert req.status_code == 200

    redirect = BeautifulSoup(req.content, features="html.parser").find('a', href=True)
    assert redirect['href'] == "%s/%s/" % (DOCS_URL, RELEASE_STREAM)


def test_latest_release_notes_redirects_correctly():
    req = requests.get("%s/latest/release-notes" % DOCS_URL)
    assert req.status_code == 200

    redirect = BeautifulSoup(req.content, features="html.parser").find('a', href=True)
    assert redirect['href'] == "%s/%s/release-notes/" % (DOCS_URL, RELEASE_STREAM)
