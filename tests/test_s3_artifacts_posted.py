import os
import requests
import yaml
import tests
from bs4 import BeautifulSoup
from parameterized import parameterized


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
HELM_CHARTS_BASE_URL = tests.HELM_CHARTS_BASE_URL
HELM_OPERATOR_URL = tests.HELM_OPERATOR_URL
ARCHIVES_BASE_URL = tests.ARCHIVES_BASE_URL
ARCHIVES_FILE_URL = tests.ARCHIVE_FILE_URL

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[0].get('title')
    HELM_RELEASE = versions[0].get('helmRelease')
    OPERATOR_RELEASE = versions[0].get('tigera-operator')['version']
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)

@parameterized({
  'operator': {
    'url': HELM_OPERATOR_URL,
  },
}.items())
def test_helm_chart_posted(name, chart):
    req_url = chart.get('url').format(
        charts_base_url=HELM_CHARTS_BASE_URL, release_version=RELEASE_VERSION, helm_release=HELM_RELEASE)
    print '[INFO] checking {0} chart is posted to {1}'.format(name, req_url)
    req = requests.head(req_url)
    assert req.status_code == 200

def test_manifests_archive_posted():
    req_url = ARCHIVES_FILE_URL.format(archives_base_url=ARCHIVES_BASE_URL, release_version=RELEASE_VERSION,
                                       operator_version=OPERATOR_RELEASE)
    req = requests.head(req_url)
    assert req.status_code == 200

