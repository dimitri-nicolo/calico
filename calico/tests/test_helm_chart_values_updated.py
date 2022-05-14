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
HELM_OPERATOR_BASE_NAME = tests.HELM_OPERATOR_BASE_NAME
HELM_OPERATOR_URL = tests.HELM_OPERATOR_URL
VALUES_FILE_NAME = 'values.yaml'

# we don't have a 1:1 naming scheme in values.yaml and versions.yml
OPERATOR_MAPPED_IMAGES = {
    'tigera-operator': 'tigeraOperator',
    'calicoctl': 'calicoctl',
    # non-tigera components
    'prometheus-operator': 'prometheusOperator',
    'prometheus-config-reloader': 'prometheusConfigReloader',
}

EXCLUDED_IMAGES = [
    # Tigera images common to both Calico and Enterprise.
    'key-cert-provisioner',

    # Calico images
    'cnx-node',
    'cnx-kube-controllers',
    'typha',
    'cnx-apiserver',
    'cnx-queryserver',
    'flexvol',
    'dikastes',

    # Enterprise images
    'calicoq',
    'ingress-collector',
    'l7-collector',
    'cloud-controllers',
    'cnx-manager',
    'voltron',
    'es-proxy',
    'fluentd',
    'fluentd-windows',
    'es-curator',
    'elastic-tsee-installer',
    'compliance-controller',
    'compliance-reporter',
    'compliance-snapshotter',
    'compliance-server',
    'compliance-benchmarker',
    'kibana',
    'intrusion-detection-controller',
    'firewall-integration',
    'license-agent',
    'tigera-cni',
    'egress-gateway',
    'guardian',
    'dex',
    'honeypod-controller',
    'honeypod',
    'honeypod-exp-service',
    'envoy-init',
    'envoy',
    'anomaly_detection_jobs',
    'elasticsearch-metrics',
    'packetcapture',
    'tigera-prometheus-service',
    'es-gateway',
    'deep-packet-inspection',
    'windows',

    # third party images
    'elasticsearch',
    'elasticsearch-operator',
    'alertmanager',
    'prometheus',
]

REGISTRY_EXCEPTION = [
    'busybox',
]

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release['title']
    HELM_RELEASE = release['helmRelease']
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)

def test_all_images_are_mapped():
  mapped_images = dict()
  mapped_images.update(OPERATOR_MAPPED_IMAGES)

  release_components = release.get('components')
  # Always include operator.
  release_components['tigera-operator'] = release.get('tigera-operator')
  version_mapped_images = {k: v for k, v in release_components.items() if v.has_key('image') and not k in EXCLUDED_IMAGES}

  assert len(mapped_images.keys()) == len(version_mapped_images.keys())
  assert set(mapped_images.keys()) == set(version_mapped_images.keys())

@parameterized({
  'operator': {
    'name': HELM_OPERATOR_BASE_NAME,
    'url': HELM_OPERATOR_URL,
    'tgz_env_var': 'HELM_OPERATOR_TGZ_OVERRIDE',
    'images': OPERATOR_MAPPED_IMAGES,
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
    print '[INFO] extracting values from path ' + '{0}/{1}'.format(chart.get('name'), VALUES_FILE_NAME)
    sub_values = tar.extractfile('{0}/charts/tigera-prometheus-operator/{1}'.format(chart.get('name'), VALUES_FILE_NAME)).read()
    print '[INFO] extracting subchart values from path ' + '{0}/charts/tigera-prometheus-operator/{1}'.format(chart.get('name'), VALUES_FILE_NAME)

    chart_values = yaml.safe_load(values)
    subchart_values = yaml.safe_load(sub_values)

    print '[INFO] compare expected/actual images & tags in the {} chart values.yaml'.format(name)
    mapped_images = chart.get('images')
    for k, v in release.get('components').items():
      if k in mapped_images.keys():
        # Search for a match in both main chart and subchart
        config = chart_values.get(mapped_images[k])
        if config is None:
            config = subchart_values.get(mapped_images[k])
        assert config != None
        expected_image = '{0}{1}'.format('' if k in REGISTRY_EXCEPTION else v.get('registry', REGISTRY) + '/', v.get('image'))
        expected_version = v.get('version')

        # Some components inside the helm chart specify a "registry" others
        # don't.
        actual_image = config.get('image')
        if 'registry' in config:
            actual_image = '{registry}/{image}'.format( registry=config.get('registry'), image=actual_image)

        # Some image tags are called "tags" while operator is called "version".
        actual_tag = None
        if 'tag' in config:
            actual_tag = config.get('tag')
        elif 'version' in config:
            actual_tag = config.get('version')

        assert actual_image == expected_image
        assert actual_tag == expected_version
