import os
import re
import requests
from bs4 import BeautifulSoup

RELEASE_STREAM = os.environ.get('RELEASE_STREAM')
REGISTRY = 'quay.io' if os.environ.get('REGISTRY') is None else os.environ.get('REGISTRY')


def test_image_registry_updated():
    req = requests.get("https://docs.tigera.io/%s/getting-started/kubernetes/installation/calico" % RELEASE_STREAM)
    assert req.status_code == 200

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Tigera Secure EE images.").find_next('code')
    images = page.text.split('\n')
    for image in images:
        if re.search('tigera', image):
            assert re.search(REGISTRY, image)
