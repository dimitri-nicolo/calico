import os
import requests
import yaml
import tests
from bs4 import BeautifulSoup


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
S3_BASE_URL = tests.S3_BASE_URL
EE_CORE_URL = tests.EE_CORE_URL
EE_URL = tests.EE_URL

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    HELM_RELEASE = versions[RELEASE_STREAM][0]['helmRelease']
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)


def test_helm_core_chart_posted():
    req_url = EE_CORE_URL.format(S3_BASE_URL, RELEASE_VERSION, HELM_RELEASE)
    print req_url
    req = requests.head(req_url)
    assert req.status_code == 200


def test_helm_ee_chart_posted():
    req_url = EE_URL.format(S3_BASE_URL, RELEASE_VERSION, HELM_RELEASE)
    print req_url
    req = requests.head(req_url)
    assert req.status_code == 200
