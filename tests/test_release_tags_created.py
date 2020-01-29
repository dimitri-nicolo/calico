import os
import requests
import yaml
import tests

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
GITHUB_API_URL = tests.GITHUB_API_URL
GITHUB_API_TOKEN = tests.GITHUB_API_TOKEN

MAPPED_COMPONENTS = {
    'cnx-manager': 'manager',
    'voltron': 'voltron',
    'cnx-apiserver': 'apiserver',
    'cnx-queryserver': 'ts-queryserver',
    'cnx-kube-controllers': 'kube-controllers-private',
    'calicoq': 'calicoq',
    'typha': 'typha-private',
    'calicoctl': 'calicoctl-private',
    'cnx-node': 'node-private',
    'dikastes': 'app-policy-private',
    'fluentd': 'fluentd-docker',
    'es-proxy': 'es-proxy-image',
    'kibana': 'kibana-docker',
    'cloud-controllers': 'cloud-controllers',
    'elastic-tsee-installer': 'intrusion-detection',
    'es-curator': 'curator',
    'intrusion-detection-controller': 'intrusion-detection',
    'compliance-controller': 'compliance',
    'compliance-reporter': 'compliance',
    'compliance-snapshotter': 'compliance',
    'compliance-server': 'compliance',
    'compliance-benchmarker': 'compliance',
    'ingress-collector': 'ingress-collector',
}

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_all_images_are_mapped():
  mapped_images = MAPPED_COMPONENTS

  version_compoments = {k: v for k, v in release.get('components').items() if v.has_key(
      'image') and not v.has_key('registry')}

  assert len(mapped_images.keys()) == len(version_compoments.keys())
  assert set(mapped_images.keys()) == set(version_compoments.keys())

def test_component_repo_has_release_branch():
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'
    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    for repo_name in set(MAPPED_COMPONENTS.values()):
      req_url = '{base_url}/repos/tigera/{repo}/branches/{branch}'.format(
          base_url=GITHUB_API_URL, repo=repo_name, branch='release-{}'.format(RELEASE_STREAM))
      res = requests.head(req_url, headers=headers)
      assert res.status_code == 200

def test_component_repo_has_release_tag():
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'
    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    for repo_name in set(MAPPED_COMPONENTS.values()):
      print repo_name
      req_url = '{base_url}/repos/tigera/{repo}/git/refs/{ref}'.format(
          base_url=GITHUB_API_URL, repo=repo_name, ref='tags/{}'.format(RELEASE_VERSION))
      res = requests.head(req_url, headers=headers)
      assert res.status_code == 200

def test_docs_repo_has_release_branch():
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'
    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    req_url = '{base_url}/repos/tigera/{repo}/branches/{branch}'.format(
        base_url=GITHUB_API_URL, repo='calico-private', branch='release-{}'.format(RELEASE_STREAM))
    res = requests.head(req_url, headers=headers)
    assert res.status_code == 200
