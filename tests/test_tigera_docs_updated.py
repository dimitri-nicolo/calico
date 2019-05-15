import os
import requests
import yaml
from bs4 import BeautifulSoup

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = os.environ.get('RELEASE_STREAM')
with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION


def test_latest_redirects_correctly():
    req = requests.get("https://docs.tigera.io/latest")
    assert req.status_code == 200

    redirect = BeautifulSoup(req.content, features="html.parser").find('a', href=True)
    assert redirect['href'] == "https://docs.tigera.io/%s/" % RELEASE_STREAM


def test_latest_releases_redirects_correctly():
    req = requests.get("https://docs.tigera.io/latest/releases")
    assert req.status_code == 200

    redirect = BeautifulSoup(req.content, features="html.parser").find('a', href=True)
    assert redirect['href'] == "https://docs.tigera.io/%s/releases/" % RELEASE_STREAM


def test_release_notes_updated():
    req = requests.get("https://docs.tigera.io/%s/releases" % RELEASE_STREAM)
    assert req.status_code == 200

    headings_txt = []
    headings = BeautifulSoup(req.content, features="html.parser").find_all('h2')
    [headings_txt.append(x.text) for x in headings]
    assert "Tigera Secure Enterprise Edition %s" % RELEASE_VERSION in headings_txt
