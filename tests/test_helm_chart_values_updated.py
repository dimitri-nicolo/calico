import os
import yaml
import tarfile
import tests
import requests
from parameterized import parameterized


# default vars
PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
REGISTRY = tests.REGISTRY
HELM_CHARTS_BASE_URL = tests.HELM_CHARTS_BASE_URL
HELM_CORE_BASE_NAME = tests.HELM_CORE_BASE_NAME
HELM_EE_BASE_NAME = tests.HELM_EE_BASE_NAME
HELM_CORE_URL = tests.HELM_CORE_URL
HELM_EE_URL = tests.HELM_EE_URL
VALUES_FILE_NAME = 'values.yaml'

# we don't have a 1:1 naming scheme in values.yaml and versions.yml
CORE_MAPPED_IMAGES = {
    'cnx-node': 'node',
    'cnx-kube-controllers': 'kubeControllers',
    'typha': 'typha',
    'cnx-apiserver': 'apiserver',
    'cnx-queryserver': 'queryserver',
    'calicoctl': 'calicoctl',
    'dikastes': 'dikastes',
    'cloud-controllers': 'cloudControllers',
}

EE_MAPPED_IMAGES = {
    'cnx-manager': 'manager',
    'voltron': 'voltron',
    'es-proxy': 'esProxy',
    'fluentd': 'fluentd',
    'es-curator': 'esCurator',
    'elastic-tsee-installer': 'elasticTseeInstaller',
    'compliance-controller': 'complianceController',
    'compliance-reporter': 'complianceReporter',
    'compliance-snapshotter': 'complianceSnapshotter',
    'compliance-server': 'complianceServer',
    'compliance-benchmarker': 'complianceBenchmarker',
    'kibana': 'kibana',
    'intrusion-detection-controller': 'intrusionDetectionController',
}

EXCLUDED_IMAGES = [
  'calicoq',
  'ingress-collector',
]

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release['title']
    HELM_RELEASE = release['helmRelease']
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)

def test_all_images_are_mapped():
  mapped_images = dict()
  mapped_images.update(CORE_MAPPED_IMAGES)
  mapped_images.update(EE_MAPPED_IMAGES)

  release_components = release.get('components')
  version_mapped_images = {k: v for k, v in release_components.items() if v.has_key('image') and v.get('image').startswith('tigera/') and not k in EXCLUDED_IMAGES}

  assert len(mapped_images.keys()) == len(version_mapped_images.keys())
  assert set(mapped_images.keys()) == set(version_mapped_images.keys())

@parameterized({
  'core': {
    'name': HELM_CORE_BASE_NAME,
    'url': HELM_CORE_URL,
    'tgz_env_var': 'HELM_CORE_TGZ_OVERRIDE',
    'images': CORE_MAPPED_IMAGES,
  },
  'ee': {
    'name': HELM_EE_BASE_NAME,
    'url': HELM_EE_URL,
    'tgz_env_var': 'HELM_EE_TGZ_OVERRIDE',
    'images': EE_MAPPED_IMAGES,
  },
}.items())
def test_chart_values_updated(name, chart):
    req_url = chart.get('url').format(
        charts_base_url=HELM_CHARTS_BASE_URL, release_version=RELEASE_VERSION, helm_release=HELM_RELEASE)

    if os.environ.get(chart.get('tgz_env_var')):
        TGZ_FILE = os.environ.get(chart.get('tgz_env_var'))
    else:
        req = requests.get(req_url, stream=True)
        assert req.status_code == 200

        # download/create a .tgz locally
        TGZ_FILE = '{}.tgz'.format(name)
        with open(TGZ_FILE, 'wb') as f:
            f.write(req.raw.read())

    # load the values.yaml file
    tar = tarfile.open(TGZ_FILE)
    values = tar.extractfile('{0}/{1}'.format(chart.get('name'), VALUES_FILE_NAME)).read()
    chart_values = yaml.safe_load(values)

    print '[INFO] compare expected/actual images & tags in the {} chart values.yaml'.format(name)
    mapped_images = chart.get('images')
    for k, v in release.get('components').items():
      if k in mapped_images.keys():
        config = chart_values.get(mapped_images[k])
        assert config != None
        assert config.get('image') == '{0}/{1}'.format(REGISTRY, v.get('image'))
        assert config.get('tag') == RELEASE_VERSION
        assert config.get('tag') == v.get('version')
