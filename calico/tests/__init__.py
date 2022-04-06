import os

# global vars
RELEASE_STREAM = os.environ.get('RELEASE_STREAM') or 'master'
DOCS_URL = os.environ.get('DOCS_URL') or 'https://docs.tigera.io'
GIT_HASH = os.environ.get('GIT_HASH') or os.popen('git rev-parse --short=10 HEAD').read().strip()

GITHUB_API_URL = 'https://api.github.com'
GITHUB_API_TOKEN = os.environ.get('GITHUB_API_TOKEN') or os.environ.get('GITHUB_ACCESS_TOKEN', 'fake-token')

# quay
QUAY_REGISTRY = 'quay.io'
QUAY_API_URL = os.environ.get('QUAY_API_URL') or 'https://{}/api/v1'.format(QUAY_REGISTRY)
QUAY_API_TOKEN = os.environ.get('QUAY_API_TOKEN') or 'fake-token'

REGISTRY = os.environ.get('REGISTRY', QUAY_REGISTRY)

# helm
HELM_CHARTS_BASE_URL = os.environ.get('HELM_CHARTS_BASE_URL', 'https://s3.amazonaws.com/tigera-public/ee/charts')
HELM_OPERATOR_BASE_NAME = os.environ.get('HELM_OPERATOR_BASE_NAME', 'tigera-operator')
HELM_OPERATOR_URL = os.environ.get('HELM_OPERATOR_URL', '{charts_base_url}/' + HELM_OPERATOR_BASE_NAME + '-{release_version}-{helm_release}.tgz')
ARCHIVES_BASE_URL = os.environ.get('ARCHIVES_BASE_URL', 'https://s3.amazonaws.com/tigera-public/ee/archives')
ARCHIVE_FILE_URL = os.environ.get('ARCHIVE_FILE_URL', '{archives_base_url}/' + 'release-{release_version}-{operator_version}.tgz')

# EE Release Branch Prefix
EE_RELEASE_BRANCH_PREFIX = os.environ.get('EE_RELEASE_BRANCH_PREFIX', 'release-calient')
