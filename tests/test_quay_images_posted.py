import os
import yaml
import requests
import tests


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
QUAY_API_TOKEN = tests.QUAY_API_TOKEN

# images not tied to a release, in _config.yml
EXCLUDED_IMAGES = ['calico-upgrade',
                   'calicoKubeControllers',
                   'configMapReload',
                   'flexvol',
                   'cni',
                   'flannel',
                   'felix',
                   'dikastes',
                   'alertManager',
                   'busybox',
                   'elasticsearchOperator',
                   'elasticsearch',
                   'prometheusOperator',
                   'prometheusConfigReloader',
                   'prometheus',
                   'cpHorizontalAutoscaler',
                   'cpVerticalAutoscaler']

# _config.yml contains latest images, need to exclude newer
# images from old releases
EXCLUDED_IMAGES_BY_VER = {'v2.3': ['compliance-snapshotter',
                      'intrusion-detection-controller',
                      'compliance-server',
                      'compliance-controller',
                      'compliance-reporter',
                      'compliance-benchmarker',
                      'kibana'],
                      'v2.4': ['compliance-benchmarker']}

# _config.yml and _data/versions.yml have diff names
VERSIONS_MAPPED_IMAGES = {'node': 'cnx-node',
                          'cloudControllers': 'cloud-controllers',
                          'kubeControllers': 'cnx-kube-controllers',
                          'cnxApiserver': 'cnx-apiserver',
                          'cnxQueryserver': 'cnx-queryserver',
                          'cnxManager': 'cnx-manager',
                          'cnxManagerProxy': 'voltron'}

# create list of images for this release
with open('%s/../_config.yml' % PATH) as f:
    images = yaml.safe_load(f)
    ALL_IMAGES = []
    [ALL_IMAGES.append(x) for x in images['imageNames'] if x not in EXCLUDED_IMAGES]

# remove any images from older releases
if EXCLUDED_IMAGES_BY_VER.has_key(RELEASE_STREAM):
    [ALL_IMAGES.remove(x) for x in EXCLUDED_IMAGES_BY_VER[RELEASE_STREAM]]

with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION

def test_release_tag_present():
    assert QUAY_API_TOKEN != 'fake-token', '[ERROR] need a real QUAY_API_TOKEN env value'

    headers = {'content-type': 'application/json', 'authorization': 'Bearer %s' % QUAY_API_TOKEN}
    for image in ALL_IMAGES:
        if image in VERSIONS_MAPPED_IMAGES:
            expected_ver = versions[RELEASE_STREAM][0]['components'][VERSIONS_MAPPED_IMAGES[image]]['version']
        else:
            expected_ver = versions[RELEASE_STREAM][0]['components'][image]['version']

        print '[INFO] checking %s:%s' % (images['imageNames'][image], expected_ver)
        req = requests.get("https://quay.io/api/v1/repository/%s/tag/%s/images" % (images['imageNames'][image], expected_ver), headers=headers)
        assert req.status_code == 200
