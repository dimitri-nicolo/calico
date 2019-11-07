import os
import re
import requests
import tests
from bs4 import BeautifulSoup

RELEASE_STREAM = tests.RELEASE_STREAM
DOCS_URL = tests.DOCS_URL
REGISTRY = tests.REGISTRY


def test_image_registry_updated():
    req = requests.get("%s/%s/getting-started/private-registry" % (DOCS_URL, RELEASE_STREAM))
    assert req.status_code == 200

    page = BeautifulSoup(req.content, features="html.parser").find("p", text="Use the following commands to pull the required Tigera Secure EE images.").find_next('code')
    images = page.text.split('\n')
    for image in images:
        if re.search('tigera', image):
            assert re.search(REGISTRY, image)
