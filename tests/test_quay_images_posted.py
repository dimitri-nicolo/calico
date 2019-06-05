import os
import yaml
import requests


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = os.environ.get('RELEASE_STREAM')
QUAY_API_TOKEN = os.environ.get('QUAY_API_TOKEN')

# images not tied to a release
EXCLUDED_IMAGES = ['calico-upgrade',
                   'calicoKubeControllers',
                   'flexvol',
                   'cni',
                   'flannel',
                   'felix',
                   'dikastes']

# _config.yml contains latest images, need to exclude newer
# images from old releases
EXCLUDED_IMAGES_BY_VER = {'v2.3': ['compliance-snapshotter',
                      'intrusion-detection-controller',
                      'compliance-server',
                      'compliance-controller',
                      'compliance-reporter']}

# _config.yml and _data/versions.yml have diff names
VERSIONS_MAPPED_IMAGES = {'node': 'cnx-node',
                          'cloudControllers': 'cloud-controllers',
                          'kubeControllers': 'cnx-kube-controllers',
                          'cnxApiserver': 'cnx-apiserver',
                          'cnxQueryserver': 'cnx-queryserver',
                          'cnxManager': 'cnx-manager',
                          'cnxManagerProxy': 'cnx-manager-proxy'}

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
    headers = {'content-type': 'application/json', 'authorization': 'Bearer %s' % QUAY_API_TOKEN}
    for image in ALL_IMAGES:
        if image in VERSIONS_MAPPED_IMAGES:
            expected_ver = versions[RELEASE_STREAM][0]['components'][VERSIONS_MAPPED_IMAGES[image]]['version']
        else:
            expected_ver = versions[RELEASE_STREAM][0]['components'][image]['version']

        print '[INFO] checking quay.io/%s:%s' % (images['imageNames'][image], expected_ver)
        req = requests.get("https://quay.io/api/v1/repository/%s/tag/%s/images" % (images['imageNames'][image], expected_ver), headers=headers)
        assert req.status_code == 200
