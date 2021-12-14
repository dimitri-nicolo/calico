#!/usr/bin/env python

import os
import shutil
from pathlib import Path


try:
    from bs4 import BeautifulSoup
except ImportError as error:
    print(" *** Please install BeautifulSoup to proceed. Use pip3 install bs4 ***")
    raise error

# env variables passed from make to script
RELEASE_DIR = os.getenv('RELEASE_DIR')
K8S_MANIFESTS = RELEASE_DIR + "/manifests"

# constants 
OCP_INSTALL_FOLDER = RELEASE_DIR + "/ocp-manifests/install-manifests"
OCP_ENTERPRISE_FOLDER = RELEASE_DIR + "/ocp-manifests/enterprise-resources"
OCP_UPGRADE_FOLDER = RELEASE_DIR + "/ocp-manifests/upgrade-manifests"

# paths are relative to the root of the repo (since we execute the file from root through Makefile)
OCP_INSTALL_DOCS = "_site/getting-started/openshift/installation/index.html"
OCP_UPGRADE_DOCS = "_site/maintenance/openshift-upgrade.html"

# manifests list
OCP_ENTERPRISE_FILES = {"tigera-enterprise-resources.yaml", "tigera-prometheus-operator.yaml", "tigera-policies.yaml"}


def parse_ocp_docs(dir_path):
    manifest_names = set()

    with open(dir_path) as f:
        soup = BeautifulSoup(f, features="html.parser")
        manifest_block = soup.body.find('div', attrs={'id': 'openshift-manifests'})

        if not manifest_block:
            raise ValueError("unable to find openshift-manifests value from html")

        lines = manifest_block.get_text(separator="/n")
        manifests_paths = lines.split("/n")
        for path in manifests_paths:
            manifest_names.add(path.split("/")[-1].strip())

    # make sure that manifests exists in the specified path
    assert len(manifest_names) > 0
    return manifest_names


def copy_to_directory(directory, manifests):
    # create the directory
    Path(directory).mkdir(parents=True, exist_ok=True)

    # copy files from k8s manifests to folder
    for dirpath, dirs, files in os.walk(K8S_MANIFESTS + "/ocp", topdown=True):
        for file in files:
            if file in manifests:
                shutil.copy2(os.path.join(dirpath, file), directory)


def create_ocp_archive():
    copy_to_directory(OCP_INSTALL_FOLDER, parse_ocp_docs(OCP_INSTALL_DOCS))
    copy_to_directory(OCP_UPGRADE_FOLDER, parse_ocp_docs(OCP_UPGRADE_DOCS))
    copy_to_directory(OCP_ENTERPRISE_FOLDER, OCP_ENTERPRISE_FILES)


if __name__ == '__main__':
    if not RELEASE_DIR:
        raise Exception('pass in a RELEASE_DIR environment variable to create manifests archive')

    create_ocp_archive()
