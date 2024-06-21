#!/usr/bin/env python3

"""
Global variables and configuration values for the test suite
"""

import argparse

from lib.utils import EnvironmentDefaults

__defaults = {
    "OPERATOR_VERSION": "v1.34.1",
    "OPERATOR_IMAGE": "tigera/operator",
    "CALICO_VERSION": "v3.19.1",
    "CHART_RELEASE": "0",
    "VERSIONS_YAML_FILE": "../../../calico/_data/versions.yml",
    "AWS_PROFILE": "helm",
    "TIGERA_S3_PUBLIC_BUCKET": "tigera-public",
    "TIGERA_GCLOUD_WINDOWS_BUCKET": "tigera-windows",
    "SELINUX_SPECFILE_PATH": "../../../selinux/calico-selinux.spec",
    "CALICO_IMAGES_SKIP": "",
    "MANIFESTS_PATH": "../../../manifests"
}

AWS_MANIFESTS_SKIP_FILES = [
    "ocp.tgz",
    "manifests.tar.gz",
    ".gitattributes",
]

SELINUX_RHEL_VERSIONS = [
    "el8",
    "el9",
]

CALIENT_UPLOADED_BINARIES = [
    "calicoctl",
    "calicoctl-darwin-amd64",
    "calicoctl-windows-amd64.exe",
    "calicoq"
]

defaults = EnvironmentDefaults(__defaults)

def show_variable_names(args):
    for variable_name in defaults.variable_names:
        print(variable_name)

def show_variables(args):
    defaults.show_variables()

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(required=True)
    parser_varnames = subparsers.add_parser("variable-names")
    parser_varnames.set_defaults(func=show_variable_names)

    parser_varvals = subparsers.add_parser("variables")
    parser_varvals.set_defaults(func=show_variables)

    args = parser.parse_args()

    args.func(args)

