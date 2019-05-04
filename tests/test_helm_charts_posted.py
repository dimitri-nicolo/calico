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


def test_helm_core_chart_posted():
    req = requests.head("https://s3.amazonaws.com/tigera-public/ee/charts/tigera-secure-ee-core-%s.tgz" % RELEASE_VERSION)
    assert req.status_code == 200


def test_helm_ee_chart_posted():
    print "https://s3.amazonaws.com/tigera-public/ee/charts/tigera-secure-ee-%s.tgz" % RELEASE_VERSION
    req = requests.head("https://s3.amazonaws.com/tigera-public/ee/charts/tigera-secure-ee-%s.tgz" % RELEASE_VERSION)
    assert req.status_code == 200
