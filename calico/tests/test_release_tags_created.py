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
    'cnx-apiserver': 'calico-private',
    'cnx-queryserver': 'ts-queryserver',
    'cnx-kube-controllers': 'calico-private',
    'calicoq': 'calico-private',
    'typha': 'calico-private',
    'calicoctl': 'calico-private',
    'cnx-node': 'calico-private',
    'dikastes': 'calico-private',
    'fluentd': 'fluentd-docker',
    'fluentd-windows': 'fluentd-docker',
    'es-proxy': 'es-proxy-image',
    'kibana': 'kibana-docker',
    'elasticsearch': 'elasticsearch-docker',
    'cloud-controllers': 'cloud-controllers',
    'elastic-tsee-installer': 'calico-private',
    'es-curator': 'curator',
    'intrusion-detection-controller': 'calico-private',
    'compliance-controller': 'compliance',
    'compliance-reporter': 'compliance',
    'compliance-snapshotter': 'compliance',
    'compliance-server': 'compliance',
    'compliance-benchmarker': 'compliance',
    'ingress-collector': 'ingress-collector',
    'l7-collector': 'l7-collector',
    'envoy-init': 'l7-collector',
    'envoy': 'envoy-docker',
    'tigera-cni': 'calico-private',
    'license-agent': 'license-agent',
    'firewall-integration': 'firewall-integration',
    'egress-gateway': 'egress-gateway',
    'guardian': 'calico-private',
    'dex': 'dexidp-docker',
    'honeypod-controller': 'calico-private',
    'anomaly_detection_jobs': 'anomaly_detection_jobs',
    'elasticsearch-metrics': 'elasticsearch-metrics',
    'packetcapture': 'calico-private',
    'tigera-prometheus-service': 'prometheus-service',
    'es-gateway': 'calico-private',
    'calico-private': 'calico-private',
    'libcalico-go-private': 'calico-private',
    'tigera-api': 'api',
    'deep-packet-inspection': 'calico-private',
    'windows': 'calico-private',
    'windows-upgrade': 'calico-private',
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

WITHOUT_IMAGES = [
    # Release items that do not have an Image
    'calico-private',
    'libcalico-go-private',
    'tigera-api',
]

WITHOUT_TAGS = [
    # Release items for which we do not publish tags yet
    # These items are ignored for tag testing but included for branch testing
    'libcalico-go-private',
    'tigera-api',
]

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_all_tigera_images_are_mapped():

    mapped_images = {k: v for k, v in MAPPED_COMPONENTS.items() if k not in WITHOUT_IMAGES }
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
    if name not in WITHOUT_TAGS:
        print '[INFO] checking {0} repo({1}) has {2} release tag'.format(name, repo_name, RELEASE_STREAM)
        headers = {'Accept': 'application/vnd.github.v3.raw', 'Authorization': 'token {}'.format(GITHUB_API_TOKEN)}
        req_url = '{base_url}/repos/tigera/{repo}/git/ref/{ref}'.format(
            base_url=GITHUB_API_URL, repo=repo_name, ref='tags/{}'.format(RELEASE_VERSION))
        res = requests.head(req_url, headers=headers)
        assert res.status_code == 200
