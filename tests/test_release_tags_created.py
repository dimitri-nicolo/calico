import os
import requests
import yaml
import tests
from parameterized import parameterized

PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
EE_RELEASE_BRANCH_PREFIX = tests.EE_RELEASE_BRANCH_PREFIX

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
    'fluentd-windows': 'fluentd-docker',
    'es-proxy': 'es-proxy-image',
    'kibana': 'kibana-docker',
    'elasticsearch': 'elasticsearch-docker',
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
    'l7-collector': 'l7-collector',
    'envoy-init': 'l7-collector',
    'envoy': 'envoy-docker',
    'tigera-cni': 'cni-plugin-private',
    'license-agent': 'license-agent',
    'firewall-integration': 'firewall-integration',
    'egress-gateway': 'egress-gateway',
    'guardian': 'voltron',
    'dex': 'dexidp-docker',
    'honeypod-controller': 'honeypod-controller',
    'anomaly_detection_jobs': 'anomaly_detection_jobs',
    'elasticsearch-metrics': 'elasticsearch-metrics',
    'packetcapture-api': 'packetcapture-api',
}

SKIP_COMPONENTS = [
    # Honeypod images aren't part of the release process.
    'honeypod',
    'honeypod-exp-service',

    # key-cert-provisioner image isn't part of the release process.
    'key-cert-provisioner',

    # third party images
    'elasticsearch-operator',
    'prometheus',
    'prometheus-operator',
    'prometheus-config-reloader',
    'configmap-reload',
    'alertmanager',
]

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_all_tigera_images_are_mapped():

    mapped_images = MAPPED_COMPONENTS

    version_components = {k: v for k, v in release.get('components').items() if v.has_key('image') and v.get('image').startswith('tigera/') and k not in SKIP_COMPONENTS}

    assert len(mapped_images.keys()) == len(version_components.keys())
    assert set(mapped_images.keys()) == set(version_components.keys())

@parameterized(MAPPED_COMPONENTS.items())
def test_component_repo_has_release_branch(name, repo_name):
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'

    release_prefix = EE_RELEASE_BRANCH_PREFIX
    if RELEASE_STREAM < 'v3.0':
        release_prefix = 'release'

    print '[INFO] checking {0} repo({1}) has {2}-{3} branch'.format(name, repo_name, release_prefix, RELEASE_STREAM)

    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    req_url = '{base_url}/repos/tigera/{repo}/branches/{branch}'.format(
        base_url=GITHUB_API_URL, repo=repo_name, branch='{0}-{1}'.format(release_prefix, RELEASE_STREAM))
    res = requests.head(req_url, headers=headers)
    assert res.status_code == 200

@parameterized(MAPPED_COMPONENTS.items())
def test_component_repo_has_release_tag(name, repo_name):
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'

    print '[INFO] checking {0} repo({1}) has {2} release tag'.format(name, repo_name, RELEASE_STREAM)

    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    req_url = '{base_url}/repos/tigera/{repo}/git/refs/{ref}'.format(
        base_url=GITHUB_API_URL, repo=repo_name, ref='tags/{}'.format(RELEASE_VERSION))
    res = requests.head(req_url, headers=headers)
    assert res.status_code == 200

def test_docs_repo_has_release_branch():
    assert GITHUB_API_TOKEN != 'fake-token', '[ERROR] need a real GITHUB_API_TOKEN env value'

    release_prefix = EE_RELEASE_BRANCH_PREFIX
    if RELEASE_STREAM < 'v3.0':
        release_prefix = 'release'

    print '[INFO] checking calico-private repo has {0}-{1} branch'.format(release_prefix, RELEASE_STREAM)

    headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
    req_url = '{base_url}/repos/tigera/{repo}/branches/{branch}'.format(
        base_url=GITHUB_API_URL, repo='calico-private', branch='{0}-{1}'.format(release_prefix, RELEASE_STREAM))
    res = requests.head(req_url, headers=headers)
    assert res.status_code == 200
