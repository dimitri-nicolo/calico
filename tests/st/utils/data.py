# Copyright (c) 2015-2016 Tigera, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Various test data that may be shared across multiple tests.
# Naming follows the approximate format:
#
# <kind>_name<idx>_rev<revision>_<key attributes>
#
# Incrementing name indexes indicate the order in which they would be listed.
#
# The rev (a-z) indicates that it should be possible to switch between different
# revisions of the same data.
#
# The key attributes provide some useful additonal data, for example (a v4 specific
# resource).
import netaddr

from utils import API_VERSION


#
# CNX Licenses
#
valid_cnx_license = {
    'apiVersion': API_VERSION,
    'kind': 'LicenseKey',
    'metadata': {
        'name': 'default'
    },
    'spec': {
        'certificate': """"
-----BEGIN CERTIFICATE-----
MIID4DCCAsigAwIBAgIRAP8sPcS0yV6BQiQ6KnHlTyAwDQYJKoZIhvcNAQELBQAw
dTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSBJbmMuMSMwIQYDVQQDExpUaWdl
cmEgSW5jLiBDZXJ0IEF1dGhvcml0eTAeFw0xODAzMjYxNjQ0MDBaFw0xOTAzMjYx
NjQ0MDBaMH0xCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYD
VQQHEw1TYW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtUaWdlcmEgSW5jLjErMCkGA1UE
AxMiVGlnZXJhIEluYy4gTGljIHNpZ25pbmcgLSBJbnRlcm5hbDCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAMYrZUYWIwv0Rcy/cQxBZEraMrJqsSOWfnst
20DFZax1FEqQDlrBxwTo4XUuyobFEYjFdluN3aEGL7TtZTN/+CbhBG2Yxb/QFHJO
SShudbE0aVPIv5u/S9cOKgqE97R8dvyS27zTE4Jm6IU8AwyqtS3+zpBvnQAwBXnH
fEzqw5ceCapkRuz3vMgxAFhv3U/OIDYdn77EGV6gmti1eX5N1YP/mEXt9HcjjL+a
CKAbq3SBg7PmDxUNbxRD0Wd7AHcSGGtgu3n7nKigrxFhXMaON035+o9QCSjanpXR
KF8AGh8sstMGXrtpGF+s8WowRizF2KWo3Tr4R5Sz7kwe6lZWAzMCAwEAAaNjMGEw
DgYDVR0PAQH/BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFG+np06E
a35Mt31j7V1kP7HMHxstMB8GA1UdIwQYMBaAFGib49CAau909ncUpB/aakasszre
MA0GCSqGSIb3DQEBCwUAA4IBAQA+tiY9jtBR15mS7a6Nz2ppHp0V5qoCClAEyYJG
gCrfZ+TeugPjxVd70N73TYqRRU66sjrX4HLFjyDYTm9H7P5EE84J40C/QpfAeA2n
TiEMgnhCxjnmaxs9JhctoKRENgrIYtTJjnL4WIJN1wuH6HLYClTSUxjS713sI9Jk
V/E7rDjAnTrgg0ZNO20/CgVSNmRgK9cjJPGA0hWAe6XquPal0FAJsfVqwQRIxXGd
TxA0EwUL8NOBHWeVY3sraVHUusOFgMiBr/pdChToD4AJ0jX5a4DzsCRSyMRVz6Dq
pdDGllpqsFfAEMF4hv4A5jhQLPk5Oz5azkoCJ0vcFCtj3nnh
-----END CERTIFICATE-----
""",
        'token': """eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiI4TTl2Nm56M09ObGRPajZKIiwidGFnIjoiUF9TSWRlZFpuRDdENS0tNnJsYUFxZyIsInR5cCI6IkpXVCJ9.O_EPhoatcbLMkxjRvBbTjw.QuN5e8twdRo7ZZUm.C-0ZkO5scQxz-CxCnDP7MuMOp7UpZoHi01NtocMjtI8mpdxRG-zKKXnDXWx3IOeanj8jCGD5PvHwbs93GjsoQu5BvnBNiOMou6EXEG9NVYZPhkqESqXBEiSQS9vt5IC-fuRX6gio0_lqAH85KJyTJcHde_ty98SbPnYytg_l3AgDSUb1-c0xdySUvYJLqk1r9tgYX05LN5wF_eOP4aznSIarLeXUBfAj16RZAO3bcOpoJjQFfAvu9XAjrr54sNxIF2j_iOqIHYm4vTo3hXgLlYVMfnhDS52DYKjs9imDF-tTSv4eRf7raAuw20xWin59zH-KQI0SV64QjPsDxKTSvpVCqyBCJpCSyLZ6qanZ8DiDhXMkHdsOikL5l3LCQKQU9iB4rzswp-ZtcFLvRysJIwxbsXh_7wAK8jdq0zgRC9isdqunbyG9DH_7LDIxBJc5cK0pdWlwFDY603H9X-WecKiiF5oF8R1-p0XCa63vFNvkkdO9_4nEbD1AHdKmLMIn6eC0bVOxImUeY2izClDueMC7EiOsONf53aZBvkhspKBNgoAOdn8mjjlj8VQ_r-D_uddgr--zb3Nm2QrHrS-Qo5J311JBCeW6SWX-pZr5qePcJyc6di4b5of4evjy9PVPZQ2hINJ7DurLLdVv_M0RbnsTElZT54Yop84nXPMcvPk8_fWcIGQSYgjzkYFX5ahERUtbT3yM43cqtMLqcJVaDYKz0sxC8x6EXh5NhFUrjiqezJfF-4Mrvr1OMXutcxrKuWq_cHAXFi2eLn68btUJjapEsYkwPkYsVmN6xKFDAa81_jPlMdfpC2kvg4hwX8rRNQ72h3aGxfBPbs9OZxEBJJw2uvNBmrFTCLhmoSc3BJnrYrw.NnXRLTAy-N_i6W05Ww-bZQ""",
    }
}

#
# IPPools
#
ippool_name1_rev1_v4 = {
    'apiVersion': API_VERSION,
    'kind': 'IPPool',
    'metadata': {
        'name': 'ippool-name1'
    },
    'spec': {
        'cidr': "10.0.1.0/24",
        'ipipMode': 'Always',
    }
}

ippool_name1_rev2_v4 = {
    'apiVersion': API_VERSION,
    'kind': 'IPPool',
    'metadata': {
        'name': 'ippool-name1'
    },
    'spec': {
        'cidr': "10.0.1.0/24",
        'ipipMode': 'Never',
    }
}

ippool_name2_rev1_v6 = {
    'apiVersion': API_VERSION,
    'kind': 'IPPool',
    'metadata': {
        'name': 'ippool-name2'
    },
    'spec': {
        'cidr': "fed0:8001::/64",
        'ipipMode': 'Never',
    }
}

#
# BGPPeers
#
bgppeer_name1_rev1_v4 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPPeer',
    'metadata': {
        'name': 'bgppeer-name-123abc',
    },
    'spec':  {
        'node': 'node1',
        'peerIP': '192.168.0.250',
        'asNumber': 64514,
    },
}

bgppeer_name1_rev2_v4 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPPeer',
    'metadata': {
        'name': 'bgppeer-name-123abc',
    },
    'spec':  {
        'node': 'node2',
        'peerIP': '192.168.0.251',
        'asNumber': 64515,
    },
}

bgppeer_name2_rev1_v6 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPPeer',
    'metadata': {
        'name': 'bgppeer-name-456def',
    },
    'spec': {
        'node': 'node2',
        'peerIP': 'fd5f::6:ee',
        'asNumber': 64590,
    },
}

#
# Tier1
#

tier_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Tier',
    'metadata': {
        'name': 'admin',
    },
    'spec': {
        'Order': 1000,
    },
}

tier_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Tier',
    'metadata': {
        'name': 'before',
    },
    'spec': {
        'Order': 100,
    },
}

#
# Network Policy
#
networkpolicy_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'NetworkPolicy',
    'metadata': {
        'name': 'default.policy-mypolicy1',
        'namespace': 'default'
    },
    'spec': {
        'tier': 'default',
        'order': 100,
        'selector': "type=='database'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Allow',
                'source': {
                    'selector': "type=='application'"},
            },
        ],
        'ingress': [
            {
                'ipVersion': 4,
                'action': 'Deny',
                'destination': {
                    'notNets': ['10.3.0.0/16'],
                    'notPorts': ['110:1050'],
                    'notSelector': "type=='apples'",
                    'nets': ['10.2.0.0/16'],
                    'ports': ['100:200'],
                    'selector': "type=='application'",
                },
                'protocol': 'TCP',
                'source': {
                    'notNets': ['10.1.0.0/16'],
                    'notPorts': [1050],
                    'notSelector': "type=='database'",
                    'nets': ['10.0.0.0/16'],
                    'ports': [1234, '10:1024'],
                    'selector': "type=='application'",
                    'namespaceSelector': 'has(role)',
                }
            }
        ],
    }
}

networkpolicy_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'NetworkPolicy',
    'metadata': {
        'name': 'policy-mypolicy1',
        'namespace': 'default'
    },
    'spec': {
        'tier': "default",
        'order': 100000,
        'selector': "type=='sql'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Deny',
                'protocol': 'TCP',
            },
        ],
        'ingress': [
            {
                'action': 'Allow',
                'protocol': 'UDP',
            },
        ],
    }
}

networkpolicy_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'NetworkPolicy',
    'metadata': {
        'name': 'default.policy-mypolicy2',
        'namespace': 'default',
        'generateName': 'test-policy-',
        'deletionTimestamp': '2006-01-02T15:04:07Z',
        'deletionGracePeriodSeconds': 30,
        'ownerReferences': [{
            'apiVersion': 'extensions/v1beta1',
            'blockOwnerDeletion': True,
            'controller': True,
            'kind': 'DaemonSet',
            'name': 'endpoint1',
            'uid': 'test-uid-change',
        }],
        'initializers': {
            'pending': [{
                'name': 'initializer1',
            }],
            'result': {
                'status': 'test-status',
            },
        },
        'clusterName': 'cluster1',
        'labels': {'label1': 'l1', 'label2': 'l2'},
        'annotations': {'key': 'value'},
        'selfLink': 'test-self-link',
        'uid': 'test-uid-change',
        'generation': 3,
        'finalizers': ['finalizer1', 'finalizer2'],
        'creationTimestamp': '2006-01-02T15:04:05Z',
    },
    'spec': {
        'tier': "default",
        'order': 100000,
        'selector': "type=='sql'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Deny',
                'protocol': 'TCP',
            },
        ],
        'ingress': [
            {
                'action': 'Allow',
                'protocol': 'UDP',
            },
        ],
    }
}

networkpolicy_tiered_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'NetworkPolicy',
    'metadata': {
        'name': 'admin.mypolicy2',
        'namespace': 'default'
    },
    'spec': {
        'tier': 'admin',
        'order': 100000,
        'selector': "type=='sql'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Deny',
                'protocol': 'TCP',
            },
        ],
        'ingress': [
            {
                'action': 'Allow',
                'protocol': 'UDP',
            },
        ],
    }
}

networkpolicy_os_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'NetworkPolicy',
    'metadata': {
        'name': 'os-policy-mypolicy1',
        'namespace': 'default'
    },
    'spec': {
        'order': 100,
        'selector': "type=='database'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Allow',
                'source': {
                    'selector': "type=='application'"},
            },
        ],
        'ingress': [
            {
                'ipVersion': 4,
                'action': 'Deny',
                'destination': {
                    'notNets': ['10.3.0.0/16'],
                    'notPorts': ['110:1050'],
                    'notSelector': "type=='apples'",
                    'nets': ['10.2.0.0/16'],
                    'ports': ['100:200'],
                    'selector': "type=='application'",
                },
                'protocol': 'TCP',
                'source': {
                    'notNets': ['10.1.0.0/16'],
                    'notPorts': [1050],
                    'notSelector': "type=='database'",
                    'nets': ['10.0.0.0/16'],
                    'ports': [1234, '10:1024'],
                    'selector': "type=='application'",
                    'namespaceSelector': 'has(role)',
                }
            }
        ],
    }
}

#
# Global Network Policy
#
globalnetworkpolicy_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkPolicy',
    'metadata': {
        'name': 'default.policy-mypolicy1',
    },
    'spec': {
        'tier': "default",
        'order': 100,
        'selector': "type=='database'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Allow',
                'source': {
                    'selector': "type=='application'"},
            },
        ],
        'ingress': [
            {
                'ipVersion': 4,
                'action': 'Deny',
                'destination': {
                    'notNets': ['10.3.0.0/16'],
                    'notPorts': ['110:1050'],
                    'notSelector': "type=='apples'",
                    'nets': ['10.2.0.0/16'],
                    'ports': ['100:200'],
                    'selector': "type=='application'",
                },
                'protocol': 'TCP',
                'source': {
                    'notNets': ['10.1.0.0/16'],
                    'notPorts': [1050],
                    'notSelector': "type=='database'",
                    'nets': ['10.0.0.0/16'],
                    'ports': [1234, '10:1024'],
                    'selector': "type=='application'",
                    'namespaceSelector': 'has(role)',
                }
            }
        ],
    }
}

globalnetworkpolicy_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkPolicy',
    'metadata': {
        'name': 'default.policy-mypolicy1',
    },
    'spec': {
        'tier': "default",
        'order': 100000,
        'selector': "type=='sql'",
        'doNotTrack': True,
        'applyOnForward': True,
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Deny',
                'protocol': 'TCP',
            },
        ],
        'ingress': [
            {
                'action': 'Allow',
                'protocol': 'UDP',
            },
        ],
    }
}

globalnetworkpolicy_tiered_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkPolicy',
    'metadata': {
        'name': 'admin.mypolicy2',
    },
    'spec': {
        'tier': 'admin',
        'order': 100000,
        'selector': "type=='sql'",
        'doNotTrack': True,
        'applyOnForward': True,
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Deny',
                'protocol': 'TCP',
            },
        ],
        'ingress': [
            {
                'action': 'Allow',
                'protocol': 'UDP',
            },
        ],
    }
}

globalnetworkpolicy_os_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkPolicy',
    'metadata': {
        'name': 'os-policy-mypolicy1',
    },
    'spec': {
        'order': 100,
        'selector': "type=='database'",
        'types': ['Ingress', 'Egress'],
        'egress': [
            {
                'action': 'Allow',
                'source': {
                    'selector': "type=='application'"},
            },
        ],
        'ingress': [
            {
                'ipVersion': 4,
                'action': 'Deny',
                'destination': {
                    'notNets': ['10.3.0.0/16'],
                    'notPorts': ['110:1050'],
                    'notSelector': "type=='apples'",
                    'nets': ['10.2.0.0/16'],
                    'ports': ['100:200'],
                    'selector': "type=='application'",
                },
                'protocol': 'TCP',
                'source': {
                    'notNets': ['10.1.0.0/16'],
                    'notPorts': [1050],
                    'notSelector': "type=='database'",
                    'nets': ['10.0.0.0/16'],
                    'ports': [1234, '10:1024'],
                    'selector': "type=='application'",
                    'namespaceSelector': 'has(role)',
                }
            }
        ],
    }
}

#
# Global network sets
#

globalnetworkset_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkSet',
    'metadata': {
        'name': 'net-set1',
    },
    'spec': {
        'nets': [
            "10.0.0.1",
            "11.0.0.0/16",
            "feed:beef::1",
            "dead:beef::96",
        ]
    }
}

# A network set with a large number of entries.  In prototyping this test, I found that there are
# "upstream" limits that cap how large we can go:
#
# - Kubernetes' gRPC API has a 4MB message size limit.
# - etcdv3 has a 1MB value size limit.
many_nets = []
for i in xrange(10000):
    many_nets.append("10.%s.%s.0/28" % (i >> 8, i % 256))
globalnetworkset_name1_rev1_large = {
    'apiVersion': API_VERSION,
    'kind': 'GlobalNetworkSet',
    'metadata': {
        'name': 'net-set1',
    },
    'spec': {
        'nets': many_nets,
    }
}

#
# Host Endpoints
#
hostendpoint_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'HostEndpoint',
    'metadata': {
        'name': 'endpoint1',
        'labels': {'type': 'database'},
    },
    'spec': {
        'interfaceName': 'eth0',
        'profiles': ['prof1', 'prof2'],
        'node': 'host1'
    }
}

hostendpoint_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'HostEndpoint',
    'metadata': {
        'name': 'endpoint1',
        'labels': {'type': 'frontend'}
    },
    'spec': {
        'interfaceName': 'cali7',
        'profiles': ['prof1', 'prof2'],
        'node': 'host2'
    }
}

hostendpoint_name1_rev3 = {
    'apiVersion': API_VERSION,
    'kind': 'HostEndpoint',
    'metadata': {
        'name': 'endpoint1',
        'labels': {'type': 'frontend', 'misc': 'version1'},
        'annotations': {'key': 'value'},
        'selfLink': 'test-self-link',
        'uid': 'test-uid-change',
        'generation': 3,
        'finalizers': ['finalizer1', 'finalizer2'],
        'creationTimestamp': '2006-01-02T15:04:05Z',
    },
    'spec': {
        'interfaceName': 'cali7',
        'profiles': ['prof1', 'prof2'],
        'node': 'host2'
    }
}

#
# Profiles
#
profile_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Profile',
    'metadata': {
        'labels': {'foo': 'bar'},
        'name': 'profile-name1'
    },
    'spec': {
        'egress': [
            {
                'action': 'Allow',
                'source': {
                      'selector': "type=='application'"
                }
            }
        ],
        'ingress': [
            {
                'ipVersion': 4,
                'action': 'Deny',
                'destination': {
                   'notNets': ['10.3.0.0/16'],
                   'notPorts': ['110:1050'],
                   'notSelector': "type=='apples'",
                   'nets': ['10.2.0.0/16'],
                   'ports': ['100:200'],
                   'selector': "type=='application'"},
                'protocol': 'TCP',
                'source': {
                   'notNets': ['10.1.0.0/16'],
                   'notPorts': [1050],
                   'notSelector': "type=='database'",
                   'nets': ['10.0.0.0/16'],
                   'ports': [1234, '10:20'],
                   'selector': "type=='application'",
                }
            }
        ],
    }
}

profile_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'Profile',
    'metadata': {
        'name': 'profile-name1',
    },
    'spec': {
        'egress': [
            {
                'action': 'Allow'
            }
        ],
        'ingress': [
            {
                'ipVersion': 6,
                'action': 'Deny',
            },
        ],
    }
}

#
# Workload Endpoints
#
workloadendpoint_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'WorkloadEndpoint',
    'metadata': {
        'labels': {
            'projectcalico.org/namespace': 'namespace1',
            'projectcalico.org/orchestrator': 'k8s',
            'type': 'database',
        },
        'name': 'node1-k8s-abcd-eth0',
        'namespace': 'namespace1',
    },
    'spec': {
        'node': 'node1',
        'orchestrator': 'k8s',
        'pod': 'abcd',
        'endpoint': 'eth0',
        'containerID': 'container1234',
        'ipNetworks': ['1.2.3.4/32'],
        'interfaceName': 'cali1234',
        'profiles': ['prof1', 'prof2'],
    }
}

workloadendpoint_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'WorkloadEndpoint',
    'metadata': {
        'labels': {
            'projectcalico.org/namespace': 'namespace1',
            'projectcalico.org/orchestrator': 'cni',
            'type': 'database'
        },
        'name': 'node2-cni-container1234-eth0',
        'namespace': 'namespace1',
    },
    'spec': {
        'node': 'node2',
        'orchestrator': 'cni',
        'endpoint': 'eth0',
        'containerID': 'container1234',
        'ipNetworks': ['1.2.3.4/32'],
        'interfaceName': 'cali1234',
        'profiles': ['prof1', 'prof2'],
    }
}

#
# Nodes
#
node_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Node',
    'metadata': {
        'name': 'node1',
    },
    'spec': {
        'bgp': {
            'ipv4Address': '1.2.3.4/24',
            'ipv6Address': 'aa:bb:cc::ff/120',
        }
    }
}

node_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Node',
    'metadata': {
        'name': 'node2',
    },
    'spec': {
        'bgp': {
            'ipv4Address': '1.2.3.5/24',
            'ipv6Address': 'aa:bb:cc::ee/120',
        }
    }
}

node_name3_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'Node',
    'metadata': {
        'name': 'node3',
    },
    'spec': {
        'bgp': {
            'ipv4Address': '1.2.3.6/24',
            'ipv6Address': 'aa:bb:cc::dd/120',
        }
    }
}

#
# BGPConfigs
#
bgpconfig_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPConfiguration',
    'metadata': {
        'name': 'default',
    },
    'spec': {
        'logSeverityScreen': 'Info',
        'nodeToNodeMeshEnabled': True,
        'asNumber': 6512,
    }
}

bgpconfig_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPConfiguration',
    'metadata': {
        'name': 'default',
    },
    'spec': {
        'logSeverityScreen': 'Info',
        'nodeToNodeMeshEnabled': False,
        'asNumber': 6511,
    }
}

bgpconfig_name2_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPConfiguration',
    'metadata': {
        'name': 'bgpconfiguration1',
    },
    'spec': {
        'logSeverityScreen': 'Info',
    }
}

bgpconfig_name2_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPConfiguration',
    'metadata': {
        'name': 'bgpconfiguration1',
    },
    'spec': {
        'logSeverityScreen': 'Debug',
    }
}

bgpconfig_name2_rev3 = {
    'apiVersion': API_VERSION,
    'kind': 'BGPConfiguration',
    'metadata': {
        'name': 'bgpconfiguration1',
    },
    'spec': {
        'logSeverityScreen': 'Debug',
        'nodeToNodeMeshEnabled': True,
    }
}

#
# Remote cluster configs
#
rcc_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'RemoteClusterConfiguration',
    'metadata': {
        'name': 'rcc1',
    },
    'spec': {
        'datastoreType': 'kubernetes',
        'kubeconfig' : 'yes- this is a valid path!'
    }
}

rcc_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'RemoteClusterConfiguration',
    'metadata': {
        'name': 'rcc1',
    },
    'spec': {
        'datastoreType': 'kubernetes',
        'kubeconfig' : '/more/normal'
    }
}

rcc_rev3 = {
    'apiVersion': API_VERSION,
    'kind': 'RemoteClusterConfiguration',
    'metadata': {
        'name': 'rcc1',
    },
    'spec': {
        'datastoreType': 'kubernetes',
        'kubeconfig' : '/etc/config/kubeconfig'
    }
}

#
# FelixConfigs
#
felixconfig_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'FelixConfiguration',
    'metadata': {
        'name': 'felixconfiguration1',
    },
    'spec': {
        'chainInsertMode': 'append',
        'defaultEndpointToHostAction': 'Accept',
        'failsafeInboundHostPorts': [
            {'protocol': 'TCP', 'port': 666},
            {'protocol': 'UDP', 'port': 333}, ],
        'failsafeOutboundHostPorts': [
            {'protocol': 'TCP', 'port': 999},
            {'protocol': 'UDP', 'port': 222},
            {'protocol': 'UDP', 'port': 422}, ],
        'interfacePrefix': 'humperdink',
        'ipipMTU': 1521,
        'ipsetsRefreshInterval': '44s',
        'iptablesFilterAllowAction': 'Return',
        'iptablesLockFilePath': '/run/fun',
        'iptablesLockProbeInterval': '500ms',
        'iptablesLockTimeout': '22s',
        'iptablesMangleAllowAction': 'Accept',
        'iptablesMarkMask': 0xff0000,
        'iptablesPostWriteCheckInterval': '12s',
        'iptablesRefreshInterval': '22s',
        'ipv6Support': True,
        'logFilePath': '/var/log/fun.log',
        'logPrefix': 'say-hello-friend',
        'logSeverityScreen': 'Info',
        'maxIpsetSize': 8192,
        'metadataAddr': '127.1.1.1',
        'metadataPort': 8999,
        'netlinkTimeout': '10s',
        'prometheusGoMetricsEnabled': True,
        'prometheusMetricsEnabled': True,
        'prometheusMetricsPort': 11,
        'prometheusProcessMetricsEnabled': True,
        'reportingInterval': '10s',
        'reportingTTL': '99s',
        'routeRefreshInterval': '33s',
        'usageReportingEnabled': False,
    }
}

felixconfig_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'FelixConfiguration',
    'metadata': {
        'name': 'felixconfiguration1',
    },
    'spec': {
        'ipv6Support': False,
        'logSeverityScreen': 'Debug',
        'netlinkTimeout': '11s',
    }
}

# The large values for `netlinkTimeout` and `reportingTTL` will be transformed
# into a different unit type in the format `XhXmXs`.
felixconfig_name1_rev3 = {
    'apiVersion': API_VERSION,
    'kind': 'FelixConfiguration',
    'metadata': {
        'name': 'felixconfiguration1',
    },
    'spec': {
        'ipv6Support': False,
        'logSeverityScreen': 'Debug',
        'netlinkTimeout': '125s',
        'reportingTTL': '9910s',
    }
}

#
# ClusterInfo
#
clusterinfo_name1_rev1 = {
    'apiVersion': API_VERSION,
    'kind': 'ClusterInformation',
    'metadata': {
        'name': 'default',
    },
    'spec': {
        'clusterGUID': 'cluster-guid1',
        'datastoreReady': True,
    }
}

clusterinfo_name1_rev2 = {
    'apiVersion': API_VERSION,
    'kind': 'ClusterInformation',
    'metadata': {
        'name': 'default',
    },
    'spec': {
        'clusterGUID': 'cluster-guid2',
        'clusterType': 'cluster-type2',
        'calicoVersion': 'calico-version2',
    }
}

