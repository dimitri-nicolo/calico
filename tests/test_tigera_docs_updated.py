import os
import requests
from bs4 import BeautifulSoup

RELEASE_STREAM = os.environ.get('RELEASE_STREAM')


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
    assert "Tigera Secure Enterprise Edition %s" % RELEASE_STREAM in headings_txt
