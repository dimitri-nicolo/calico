import os
import yaml
import tarfile
import tests
import requests


# default vars
PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
S3_BASE_URL = tests.S3_BASE_URL
EE_CORE_URL = tests.EE_CORE_URL
EE_URL = tests.EE_URL

# we don't have a 1:1 naming scheme in values.yaml and versions.yml
CORE_MAPPED_IMAGES = {'cnxApiserver': 'apiserver',
                      'cnxQueryserver': 'queryserver',
                      'node': 'node',
                      'dikastes': 'dikastes',
                      'calicoctl': 'calicoctl',
                      'typha': 'typha',
                      'kubeControllers': 'kubeControllers',
                      'cloudControllers': 'cloudControllers'}
                      
VERSIONS_MAPPED_IMAGES = {'node': 'cnx-node',
                          'cloudControllers': 'cloud-controllers',
                          'kubeControllers': 'cnx-kube-controllers',
                          'cnxApiserver': 'cnx-apiserver',
                          'cnxQueryserver': 'cnx-queryserver',
                          'cnxManager': 'cnx-manager',
                          'cnxManagerProxy': 'cnx-manager-proxy'}

EE_MAPPED_IMAGES = {'intrusion-detection-controller': 'intrusionDetectionController',
                    'cnxManager': 'manager',
                    'cnxManagerProxy': 'managerProxy',
                    'es-proxy': 'esProxy',
                    'fluentd': 'fluentd',
                    'es-curator': 'esCurator',
                    'elastic-tsee-installer': 'elasticTseeInstaller',
                    'compliance-controller': 'complianceController',
                    'compliance-server': 'complianceServer',
                    'compliance-snapshotter': 'complianceSnapshotter',
                    'compliance-reporter': 'complianceReporter'}

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    HELM_RELEASE = versions[RELEASE_STREAM][0]['helmRelease']
    print '[INFO] using _data/versions.yaml, discovered version: {0}-{1}'.format(RELEASE_VERSION, HELM_RELEASE)


def test_core_chart_values_updated():
    req_url = EE_CORE_URL.format(S3_BASE_URL, RELEASE_VERSION, HELM_RELEASE)
    req = requests.get(req_url, stream=True)
    assert req.status_code == 200

    # download/create a .tgz locally
    with open("core.tgz", 'wb') as f:
        f.write(req.raw.read())

    # load the values.yaml file
    tar = tarfile.open('core.tgz')
    values = tar.extractfile('tigera-secure-ee-core/values.yaml').read()
    core_values = yaml.safe_load(values)

    # compare expected/actual imageNames:tag in the chart values.yaml
    with open('%s/../_config.yml' % PATH) as f:
        config_images = yaml.safe_load(f)
        for config_image in config_images['imageNames']:
            if config_image in CORE_MAPPED_IMAGES:
                if config_image in VERSIONS_MAPPED_IMAGES:
                    expected_ver = versions[RELEASE_STREAM][0]['components'][VERSIONS_MAPPED_IMAGES[config_image]]['version']
                else:
                    expected_ver = versions[RELEASE_STREAM][0]['components'][config_image]['version']

                expected_image = 'quay.io/' + config_images['imageNames'][config_image] + ':%s' % expected_ver
                image_path = core_values[CORE_MAPPED_IMAGES[config_image]]['image']
                image_tag = core_values[CORE_MAPPED_IMAGES[config_image]]['tag']

                print expected_image
                assert expected_image == image_path + ':' + image_tag

def test_ee_chart_values_updated():
    req_url = EE_URL.format(S3_BASE_URL, RELEASE_VERSION, HELM_RELEASE)
    req = requests.get(req_url, stream=True)
    assert req.status_code == 200

    # download/create a .tgz locally
    with open("ee.tgz", 'wb') as f:
        f.write(req.raw.read())

    # load the values.yaml file
    tar = tarfile.open('ee.tgz')
    values = tar.extractfile('tigera-secure-ee/values.yaml').read()
    core_values = yaml.safe_load(values)

    # compare expected/actual imageNames:tag in the chart values.yaml
    with open('%s/../_config.yml' % PATH) as f:
        config_images = yaml.safe_load(f)
        for config_image in config_images['imageNames']:
            if config_image in EE_MAPPED_IMAGES:
                if config_image in VERSIONS_MAPPED_IMAGES:
                    expected_ver = versions[RELEASE_STREAM][0]['components'][VERSIONS_MAPPED_IMAGES[config_image]]['version']
                else:
                    expected_ver = versions[RELEASE_STREAM][0]['components'][config_image]['version']

                expected_image = 'quay.io/' + config_images['imageNames'][config_image] + ':%s' % expected_ver
                image_path = core_values[EE_MAPPED_IMAGES[config_image]]['image']
                image_tag = core_values[EE_MAPPED_IMAGES[config_image]]['tag']

                print expected_image
                assert expected_image == image_path + ':' + image_tag
