#!/usr/bin/env python2
"""Generate Workload Endpoints

Usage:
  generate_resources.py (wep|hep) [--num=<num> --per-node=<num> --offset=<num>]
  generate_resources.py (rcc|profile) [--num=<num> --offset=<num> --ds=<datastore>]

Options:
  -h --help         Show this screen.
  --num=<num>       The number of resources to generate [default: 1].
  --per-node=<num>  The number of resources to place per node [default: 99999]
  --offset=<num>    The number to start counting from. [default: 1]
  --ds=<datastore>  The backend type to use. Either etcdv3 or kubernetes [default: etcdv3]
"""

import ipaddress
import pyaml
from docopt import docopt


def generate_hep():
    address = ipaddress.ip_address('10.244.0.0')

    for x in range(offset, num + offset):
        print pyaml.dump({
            "apiVersion": "projectcalico.org/v3",
            "kind": "HostEndpoint",
            "metadata": {
                "labels": {
                    "app": "x-{}-hep".format(x)
                },
                "name": "hep-{}".format(x)
            },
            "spec": {
                "interfaceName": "eth0",
                "node": "node{}".format(int(x / per_node)),
                "expectedIPs": [
                    str(address)
                ]
            }
        }, explicit_start=True, explicit_end=True)
        address += 1


def generate_wep():
    address = ipaddress.ip_address('10.244.0.0')

    for x in range(offset, num + offset):
        if ds_type == "etcdv3":
            print pyaml.dump({
                "apiVersion": "projectcalico.org/v3",
                "kind": "WorkloadEndpoint",
                "metadata": {
                    "labels": {
                        "app": "x-{}-wep".format(x),
                        "projectcalico.org/namespace": "default",
                        "projectcalico.org/orchestrator": "k8s",
                        "namespace": "default",
                    },
                    "name": "node{}-k8s-wep--{}-eth0".format(int(x / per_node), x),
                },
                "spec": {
                    "interfaceName": "eth0",
                    "node": "node{}".format(int(x / per_node)),
                    "ipNetworks": [
                        str(address)
                    ],
                    "containerID": "13374955926535",
                    "pod": "wep-{}".format(x),
                    "endpoint": "eth0",
                    "orchestrator": "k8s",
                    "interfaceName": "cali0ef24ba",
                    "mac": "ca:fe:1d:52:bb:e9",
                    "profiles": ["profile{}".format(int(x / per_node))],
                }
            }, explicit_start=True, explicit_end=True)
        address += 1


def generate_profile():
    for x in range(offset, num + offset):
        if ds_type == "etcdv3":
            print pyaml.dump({
                "apiVersion": "projectcalico.org/v3",
                "kind": "Profile",
                "metadata": {
                    "name": "profile{}".format(x)
                },
                "spec": {
                    "labelsToApply": {
                        "fixed": "label",
                        "profileID": "profile{}".format(x),
                    }
                }
            }, explicit_start=True, explicit_end=True)
        else:
            # For K8s, create a namespace. These appear to Calico as profiles.
            print pyaml.dump({
                "apiVersion": "v1",
                "kind": "Namespace",
                "metadata": {
                    "name": "profile{}".format(x)
                },
            }, explicit_start=True, explicit_end=True)


def generate_rcc():
    for x in range(offset, num + offset):
        resource = {
            "apiVersion": "projectcalico.org/v3",
            "kind": "RemoteClusterConfiguration",
            "metadata": {
                "name": "rcc-{}-{}".format(ds_type, x)
            },
            "spec": {
                "DatastoreType": ds_type,
            }
        }
        if ds_type == "etcdv3":
            resource["spec"]["EtcdEndpoints"] = "http://127.0.0.1:{}".format(x + 2380)
        else:
            resource["spec"]["K8sAPIEndpoint"] = "http://127.0.0.1:{}".format(x + 8079)
        print pyaml.dump(resource, explicit_start=True, explicit_end=True)


if __name__ == '__main__':
    arguments = docopt(__doc__)
    num = int(arguments["--num"])
    per_node = int(arguments["--per-node"])
    offset = int(arguments["--offset"])
    ds_type = arguments["--ds"]

    if arguments["hep"]:
        generate_hep()
    elif arguments["wep"]:
        generate_wep()
    elif arguments["rcc"]:
        generate_rcc()
    elif arguments["profile"]:
        generate_profile()
    else:
        print "How did you get here?"
