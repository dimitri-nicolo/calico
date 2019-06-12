import os

# global vars
RELEASE_STREAM = os.environ.get('RELEASE_STREAM') or 'master'
DOCS_URL = os.environ.get('DOCS_URL') or 'https://docs.tigera.io'
GIT_HASH = os.environ.get('GIT_HASH') or os.popen('git rev-parse --short HEAD').read().strip()
REGISTRY = os.environ.get('REGISTRY') or 'quay.io'
QUAY_API_TOKEN = os.environ.get('QUAY_API_TOKEN') or 'fake-token'

# helm
S3_BASE_URL = os.environ.get('S3_BASE_URL') or "https://s3.amazonaws.com/tigera-public/ee/charts"
EE_CORE_URL = os.environ.get('EE_CORE_URL') or "{0}/tigera-secure-ee-core-{1}-{2}.tgz"
EE_URL = os.environ.get('EE_URL') or "{0}/tigera-secure-ee-{1}-{2}.tgz"
