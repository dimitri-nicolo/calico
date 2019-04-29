import os
import yaml
import requests


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = os.environ.get('RELEASE_STREAM')
QUAY_API_TOKEN = os.environ.get('QUAY_API_TOKEN')
EXCLUDED_IMAGES = ['calico/upgrade',
                   'calico/kube-controllers',
                   'quay.io/calico/pod2daemon-flexvol',
                   'quay.io/calico/cni',
                   'quay.io/coreos/flannel',
                   'tigera/felix',
                   'tigera/dikastes']


with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    RELEASE_VERSION = versions[RELEASE_STREAM][0]['title']
    print '[INFO] using _data/versions.yaml, discovered version: %s' % RELEASE_VERSION


def test_release_tag_present():
    with open('%s/../_config.yml' % PATH) as f:
        images = yaml.safe_load(f)
        for image in images['imageNames']:
            if images['imageNames'][image] not in EXCLUDED_IMAGES:
                headers = {'content-type': 'application/json', 'authorization': 'Bearer %s' % QUAY_API_TOKEN}
                print '[INFO] checking quay.io/%s:%s' % (images['imageNames'][image], RELEASE_VERSION)

                req = requests.get("https://quay.io/api/v1/repository/%s/tag/%s/images" % (images['imageNames'][image], RELEASE_VERSION), headers=headers)
                assert req.status_code == 200
