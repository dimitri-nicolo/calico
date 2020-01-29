import os

# global vars
RELEASE_STREAM = os.environ.get('RELEASE_STREAM', 'master')
DOCS_URL = os.environ.get('DOCS_URL', 'https://docs.tigera.io')
GIT_HASH = os.environ.get('GIT_HASH', os.popen('git rev-parse --short=9 HEAD').read().strip())

GITHUB_API_URL = 'https://api.github.com'
GITHUB_API_TOKEN = os.environ.get('GITHUB_API_TOKEN', os.environ.get('GITHUB_ACCESS_TOKEN', 'fake-token'))

# quay
QUAY_REGISTRY = 'quay.io'
QUAY_API_URL = os.environ.get('QUAY_API_URL', 'https://{}/api/v1'.format(QUAY_REGISTRY))
QUAY_API_TOKEN = os.environ.get('QUAY_API_TOKEN', 'fake-token')

REGISTRY = os.environ.get('REGISTRY', QUAY_REGISTRY)

# helm
HELM_CHARTS_BASE_URL = os.environ.get('HELM_CHARTS_BASE_URL', 'https://s3.amazonaws.com/tigera-public/ee/charts')
HELM_CORE_BASE_NAME = os.environ.get('HELM_CORE_BASE_NAME', 'tigera-secure-ee-core')
HELM_EE_BASE_NAME = os.environ.get('HELM_CORE_BASE_NAME', 'tigera-secure-ee')
HELM_CORE_URL = os.environ.get('HELM_CORE_URL') or '{charts_base_url}/' + HELM_CORE_BASE_NAME + '-{release_version}-{helm_release}.tgz'
HELM_EE_URL = os.environ.get('HELM_EE_URL') or '{charts_base_url}/' + HELM_EE_BASE_NAME + '-{release_version}-{helm_release}.tgz'
