import os
import yaml
import requests
import tests
from parameterized import parameterized


PATH = os.path.abspath(os.path.dirname(__file__))
RELEASE_STREAM = tests.RELEASE_STREAM
QUAY_REGISTRY = tests.QUAY_REGISTRY
QUAY_API_URL = tests.QUAY_API_URL
QUAY_API_TOKEN = tests.QUAY_API_TOKEN

COMPONENTS_WITH_TESLA_IMAGES=[
    'cnx-manager',
    'kibana',
]

_TESLA_TAG_PREFIX = 'tesla'

# create list of images for this release
with open('%s/../_data/versions.yml' % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get('title')
    print '[INFO] using _data/versions.yaml, discovered version: {0}'.format(RELEASE_VERSION)

    VERSION_MAPPED_IMAGES={}

    # add all components from the versions.yml file that declares an image field
    for k,v in release.get('components').items():
        if v.has_key('image'):
            VERSION_MAPPED_IMAGES.update({k:v})

    # add tigera-operator to VERSION_MAPPED_IMAGES to check tigera-operator release tag
    VERSION_MAPPED_IMAGES.update({'tigera-operator':release.get('tigera-operator')})
    headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer {}'.format(QUAY_API_TOKEN)}


@parameterized(VERSION_MAPPED_IMAGES.items())
def test_release_images_are_limited_to_tigera_registries(component, details):
    assert details.get('image').startswith('tigera/') or details.get('image').startswith('calico/')
    if details.has_key('registry'):
        assert details.get('registry') == 'quay.io'


@parameterized(VERSION_MAPPED_IMAGES.items())
def test_release_tag_present(name, component):
    assert QUAY_API_TOKEN != 'fake-token', '[ERROR] need a real QUAY_API_TOKEN env value'

    def check_image(name, image_name, expected_ver):
        req_url = '{base_url}/repository/{image_name}/tag'.format(
            base_url=QUAY_API_URL, image_name=image_name)
        params = {'specificTag': expected_ver}
        res = requests.get(req_url, headers=headers, params=params)
        assert res.status_code == 200
        tags = res.json()['tags']
        found = False
        for tag in tags:
            if tag['name'] == expected_ver:
                found = True
                break
        assert found == True

    image_name = component.get('image')
    expected_ver = component.get('version')
    check_image(name, image_name, expected_ver)

    if name in COMPONENTS_WITH_TESLA_IMAGES:
        expected_ver = component.get('version')
        check_image(name, image_name,
                    '{prefix}-{tag}'.format(prefix=_TESLA_TAG_PREFIX, tag=expected_ver))
