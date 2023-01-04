import os
import yaml
import requests
import tarfile
from nose.tools import with_setup


PATH = os.path.abspath(os.path.dirname(__file__))
with open("%s/../_data/versions.yml" % PATH) as f:
    versions = yaml.safe_load(f)
    release = versions[0]
    RELEASE_VERSION = release.get("title")
    OPERATOR_VERSION = release.get("tigera-operator").get("version")
    ARCHIVE_NAME = "{}-{}".format(RELEASE_VERSION, OPERATOR_VERSION)

url = "https://downloads.tigera.io/ee/archives/{a}.tgz".format(a=ARCHIVE_NAME)

manifest_list = [
    "calico-windows-bgp.yaml",
    "calico-windows-vxlan.yaml",
    "calicoctl.yaml",
    "tigera-operator.yaml",
    "tigera-prometheus-operator.yaml",
    "custom-resources.yaml",
    "custom-resources-upgrade-from-calico.yaml",
    "tigera-policies.yaml",
    "aks/tigera-operator-upgrade.yaml",
    "aks/custom-resources.yaml",
    "aks/custom-resources-calico-cni.yaml",
    "aks/custom-resources-upgrade-from-calico.yaml",
    "eks/custom-resources.yaml",
    "eks/custom-resources-calico-cni.yaml",
    "eks/custom-resources-upgrade-from-calico.yaml",
    "rancher/custom-resources-rke2.yaml",
]

def setup_archive():
    response = requests.get(url, stream=True)
    global file
    file = tarfile.open(fileobj=response.raw, mode="r|gz")


def teardown_archive():
    file.close()


@with_setup(setup=setup_archive, teardown=teardown_archive)
def test_manifest_present():
    for manifest in manifest_list:
        yield check_manifest_present, manifest


def check_manifest_present(manifest):
    print("[INFO] checking {} is in archive".format(manifest))
    try:
        manifest_info = file.getmember(
            "release-{v}/manifests/{m}".format(v=RELEASE_VERSION, m=manifest)
        )
        print(manifest_info.name, manifest_info.size)
        assert manifest_info.isfile()
        assert manifest_info.size > 100
    except KeyError:
        assert False, "{m} not found in archive: {url}".format(m=manifest, url=url)
