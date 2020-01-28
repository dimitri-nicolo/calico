import os
import requests
import yaml
import tests
from bs4 import BeautifulSoup


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
HELM_CHARTS_BASE_URL = tests.HELM_CHARTS_BASE_URL
HELM_CORE_URL = tests.HELM_CORE_URL
HELM_EE_URL = tests.HELM_EE_URL

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[0].get('title')
    HELM_RELEASE = versions[0].get('helmRelease')
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)


def test_helm_core_chart_posted():
    req_url = HELM_CORE_URL.format(
        charts_base_url=HELM_CHARTS_BASE_URL, release_version=RELEASE_VERSION, helm_release=HELM_RELEASE)
    print req_url
    req = requests.head(req_url)
    assert req.status_code == 200


def test_helm_ee_chart_posted():
    req_url = HELM_EE_URL.format(
        charts_base_url=HELM_CHARTS_BASE_URL, release_version=RELEASE_VERSION, helm_release=HELM_RELEASE)
    print req_url
    req = requests.head(req_url)
    assert req.status_code == 200
