## Global Network Policy

### Create

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "fF2sfmoBmcPPPMIsGY0v",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:09:23Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:09:23Z",
    "auditID": "8cad7509-1f4a-44f7-a823-4b97d2b41ba6",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/globalnetworkpolicies",
    "verb": "create",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "globalnetworkpolicies",
      "name": "default.test-audit-global-policy",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3"
    },
    "responseStatus": {
      "metadata": {},
      "code": 201
    },
    "requestObject": {
      "kind": "GlobalNetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-global-policy",
        "creationTimestamp": null
      },
      "spec": {
        "tier": "default",
        "order": 1200,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "",
        "types": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "GlobalNetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-global-policy",
        "selfLink": "/apis/projectcalico.org/v3/globalnetworkpolicies/default.test-audit-global-policy",
        "uid": "32a02561-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520478",
        "creationTimestamp": "2019-05-03T17:09:23Z",
        "labels": {
          "projectcalico.org/tier": "default"
        }
      },
      "spec": {
        "tier": "default",
        "order": 1200,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "",
        "types": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:09:23.174502Z",
    "stageTimestamp": "2019-05-03T17:09:23.188614Z",
    "name": "default.test-audit-global-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:09:23.174Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:09:23.188Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:09:23.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:09:23.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:09:23.000Z"
    ]
  },
  "sort": [
    1556903363000
  ]
}
```

### Update

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "VF2vfmoBmcPPPMIsVY7t",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:12:54Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:12:54Z",
    "auditID": "6dec5b37-075d-4bc9-8292-120a2b8b506b",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/globalnetworkpolicies/default.test-audit-global-policy",
    "verb": "update",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "globalnetworkpolicies",
      "name": "default.test-audit-global-policy",
      "uid": "32a02561-6dc6-11e9-af27-42010a8000a7",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3",
      "resourceVersion": "520478"
    },
    "responseStatus": {
      "metadata": {},
      "code": 200
    },
    "requestObject": {
      "kind": "GlobalNetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-global-policy",
        "uid": "32a02561-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520478",
        "creationTimestamp": null
      },
      "spec": {
        "tier": "default",
        "order": 1200,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"connect-test\"||app == \"test-connect\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "GlobalNetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-global-policy",
        "selfLink": "/apis/projectcalico.org/v3/globalnetworkpolicies/default.test-audit-global-policy",
        "uid": "32a02561-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520778",
        "creationTimestamp": "2019-05-03T17:09:23Z",
        "labels": {
          "projectcalico.org/tier": "default"
        }
      },
      "spec": {
        "tier": "default",
        "order": 1200,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"connect-test\"||app == \"test-connect\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:12:54.262677Z",
    "stageTimestamp": "2019-05-03T17:12:54.277952Z",
    "name": "default.test-audit-global-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:12:54.262Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:12:54.277Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:12:54.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:09:23.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:12:54.000Z"
    ]
  },
  "sort": [
    1556903574000
  ]
}
```

### Delete

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "0V2wfmoBmcPPPMIsko6n",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:14:15Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:14:15Z",
    "auditID": "f1a23a20-874f-4487-9f23-324858654c3c",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/globalnetworkpolicies/default.test-audit-global-policy",
    "verb": "delete",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "globalnetworkpolicies",
      "name": "default.test-audit-global-policy",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3"
    },
    "responseStatus": {
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "default.test-audit-global-policy",
        "group": "projectcalico.org",
        "kind": "globalnetworkpolicies",
        "uid": "32a02561-6dc6-11e9-af27-42010a8000a7"
      },
      "code": 200
    },
    "responseObject": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "default.test-audit-global-policy",
        "group": "projectcalico.org",
        "kind": "globalnetworkpolicies",
        "uid": "32a02561-6dc6-11e9-af27-42010a8000a7"
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:14:15.592714Z",
    "stageTimestamp": "2019-05-03T17:14:15.608424Z",
    "name": null
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:14:15.592Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:14:15.608Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:14:15.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:14:15.000Z"
    ]
  },
  "highlight": {
    "verb": [
      "@kibana-highlighted-field@delete@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556903655000
  ]
}
```

# Network Policy

### Create

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "P12rfmoBmcPPPMIsu40-",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:08:58Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:08:58Z",
    "auditID": "12017d4b-b66e-4d3b-a7bf-d3cd6de1da7d",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/namespaces/default/networkpolicies",
    "verb": "create",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "default.test-audit-policy",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3"
    },
    "responseStatus": {
      "metadata": {},
      "code": 201
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-policy",
        "namespace": "default",
        "creationTimestamp": null
      },
      "spec": {
        "tier": "default",
        "order": 1100,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"test-connect\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-policy",
        "namespace": "default",
        "selfLink": "/apis/projectcalico.org/v3/namespaces/default/networkpolicies/default.test-audit-policy",
        "uid": "24260304-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520443/",
        "creationTimestamp": "2019-05-03T17:08:58Z",
        "labels": {
          "projectcalico.org/tier": "default"
        }
      },
      "spec": {
        "tier": "default",
        "order": 1100,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"test-connect\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:08:58.886183Z",
    "stageTimestamp": "2019-05-03T17:08:58.901546Z",
    "name": "default.test-audit-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:08:58.886Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:08:58.901Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:08:58.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:08:58.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:08:58.000Z"
    ]
  },
  "sort": [
    1556903338000
  ]
}
```

### Update

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "O12vfmoBmcPPPMIsC45k",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:12:35Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:12:35Z",
    "auditID": "522e974d-c551-4afb-8518-ef1119899c32",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/namespaces/default/networkpolicies/default.test-audit-policy",
    "verb": "update",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "default.test-audit-policy",
      "uid": "24260304-6dc6-11e9-af27-42010a8000a7",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3",
      "resourceVersion": "520443"
    },
    "responseStatus": {
      "metadata": {},
      "code": 200
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-policy",
        "namespace": "default",
        "uid": "24260304-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520443",
        "creationTimestamp": null
      },
      "spec": {
        "tier": "default",
        "order": 1100,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"test-connect\"||app == \"connect-test\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "default.test-audit-policy",
        "namespace": "default",
        "selfLink": "/apis/projectcalico.org/v3/namespaces/default/networkpolicies/default.test-audit-policy",
        "uid": "24260304-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520752/",
        "creationTimestamp": "2019-05-03T17:08:58Z",
        "labels": {
          "projectcalico.org/tier": "default"
        }
      },
      "spec": {
        "tier": "default",
        "order": 1100,
        "ingress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {}
          }
        ],
        "selector": "app == \"test-connect\"||app == \"connect-test\"",
        "types": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:12:35.627769Z",
    "stageTimestamp": "2019-05-03T17:12:35.639719Z",
    "name": "default.test-audit-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:12:35.627Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:12:35.639Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:12:35.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:08:58.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:12:35.000Z"
    ]
  },
  "sort": [
    1556903555000
  ]
}
```

### Delete

```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190503",
  "_type": "fluentd",
  "_id": "u12wfmoBmcPPPMIsd441",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:14:08Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:14:08Z",
    "auditID": "c4c48203-55b3-43e4-a9a9-ce64b7283f89",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/namespaces/default/networkpolicies/default.test-audit-policy",
    "verb": "delete",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "default.test-audit-policy",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3"
    },
    "responseStatus": {
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "default.test-audit-policy",
        "group": "projectcalico.org",
        "kind": "networkpolicies",
        "uid": "24260304-6dc6-11e9-af27-42010a8000a7"
      },
      "code": 200
    },
    "responseObject": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "default.test-audit-policy",
        "group": "projectcalico.org",
        "kind": "networkpolicies",
        "uid": "24260304-6dc6-11e9-af27-42010a8000a7"
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:14:08.656319Z",
    "stageTimestamp": "2019-05-03T17:14:08.672552Z",
    "name": null
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:14:08.656Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:14:08.672Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:14:08.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:14:08.000Z"
    ]
  },
  "highlight": {
    "verb": [
      "@kibana-highlighted-field@delete@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556903648000
  ]
}
```

# Kubernetes Network Policy - v1

### Create

```
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "312tfmoBmcPPPMIsrY2c",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:11:06Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:11:06Z",
    "auditID": "afef1ad8-f8a5-43aa-a58d-fa9586c104ea",
    "stage": "ResponseComplete",
    "requestURI": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies",
    "verb": "create",
    "user": {
      "username": "jane",
      "uid": "1",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "apiGroup": "networking.k8s.io",
      "apiVersion": "v1"
    },
    "responseStatus": {
      "metadata": {},
      "code": 201
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "creationTimestamp": null
      },
      "spec": {
        "podSelector": {
          "matchLabels": {
            "app": "connect-test"
          }
        },
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "selfLink": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
        "uid": "6ff57ab1-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520624",
        "generation": 1,
        "creationTimestamp": "2019-05-03T17:11:06Z"
      },
      "spec": {
        "podSelector": {
          "matchLabels": {
            "app": "connect-test"
          }
        },
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:11:06.080548Z",
    "stageTimestamp": "2019-05-03T17:11:06.090802Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": "RBAC: allowed by ClusterRoleBinding \"jane-tigera\" of ClusterRole \"network-admin\" to User \"jane\""
    },
    "name": "test-audit-k8s-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:11:06.080Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:11:06.090Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:11:06.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:11:06.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:11:06.000Z"
    ]
  },
  "highlight": {
    "objectRef.name": [
      "@kibana-highlighted-field@test@/kibana-highlighted-field@-@kibana-highlighted-field@audit@/kibana-highlighted-field@-@kibana-highlighted-field@k8s@/kibana-highlighted-field@-@kibana-highlighted-field@policy@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556903466000
  ]
}
```

### Update

```
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "EV2ufmoBmcPPPMIsaY7J",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:11:55Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:11:55Z",
    "auditID": "4be3ba15-0cc6-4fd9-b5d7-ff3eb8d42e9f",
    "stage": "ResponseComplete",
    "requestURI": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
    "verb": "update",
    "user": {
      "username": "jane",
      "uid": "1",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "uid": "6ff57ab1-6dc6-11e9-af27-42010a8000a7",
      "apiGroup": "networking.k8s.io",
      "apiVersion": "v1",
      "resourceVersion": "520624"
    },
    "responseStatus": {
      "metadata": {},
      "code": 200
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "uid": "6ff57ab1-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520624",
        "creationTimestamp": null
      },
      "spec": {
        "podSelector": {
          "matchLabels": {
            "app": "test-connect"
          }
        },
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "selfLink": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
        "uid": "6ff57ab1-6dc6-11e9-af27-42010a8000a7",
        "resourceVersion": "520695",
        "generation": 2,
        "creationTimestamp": "2019-05-03T17:11:06Z"
      },
      "spec": {
        "podSelector": {
          "matchLabels": {
            "app": "test-connect"
          }
        },
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:11:55.684168Z",
    "stageTimestamp": "2019-05-03T17:11:55.687872Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": "RBAC: allowed by ClusterRoleBinding \"jane-tigera\" of ClusterRole \"network-admin\" to User \"jane\""
    },
    "name": "test-audit-k8s-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:11:55.684Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:11:55.687Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:11:55.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T17:11:06.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:11:55.000Z"
    ]
  },
  "highlight": {
    "objectRef.name": [
      "@kibana-highlighted-field@test@/kibana-highlighted-field@-@kibana-highlighted-field@audit@/kibana-highlighted-field@-@kibana-highlighted-field@k8s@/kibana-highlighted-field@-@kibana-highlighted-field@policy@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556903515000
  ]
}
```

### Delete

```
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "sl2wfmoBmcPPPMIsVI6D",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T17:14:04Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T17:14:04Z",
    "auditID": "2cdc0922-06e1-4c19-9100-b8b5f6c4ea15",
    "stage": "ResponseComplete",
    "requestURI": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
    "verb": "delete",
    "user": {
      "username": "jane",
      "uid": "1",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "apiGroup": "networking.k8s.io",
      "apiVersion": "v1"
    },
    "responseStatus": {
      "metadata": {},
      "status": "Success",
      "code": 200
    },
    "responseObject": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "test-audit-k8s-policy",
        "group": "networking.k8s.io",
        "kind": "networkpolicies",
        "uid": "6ff57ab1-6dc6-11e9-af27-42010a8000a7"
      }
    },
    "requestReceivedTimestamp": "2019-05-03T17:14:04.413279Z",
    "stageTimestamp": "2019-05-03T17:14:04.418199Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": "RBAC: allowed by ClusterRoleBinding \"jane-tigera\" of ClusterRole \"network-admin\" to User \"jane\""
    },
    "name": null
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T17:14:04.413Z"
    ],
    "stageTimestamp": [
      "2019-05-03T17:14:04.418Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T17:14:04.000Z"
    ],
    "timestamp": [
      "2019-05-03T17:14:04.000Z"
    ]
  },
  "highlight": {
    "verb": [
      "@kibana-highlighted-field@delete@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556903644000
  ]
}
```

### Kubernetes Network Policy - extensions/v1beta1

```
Create
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "3V3gfmoBmcPPPMIsdpxp",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T18:06:39Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T18:06:39Z",
    "auditID": "6d98b223-7f2d-4647-8e07-7ed443b47d4d",
    "stage": "ResponseComplete",
    "requestURI": "/apis/extensions/v1beta1/namespaces/default/networkpolicies",
    "verb": "create",
    "user": {
      "username": "kubernetes-admin",
      "groups": [
        "system:masters",
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "199.116.73.11"
    ],
    "userAgent": "kubectl/v1.12.1 (linux/amd64) kubernetes/4ed3216",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "apiGroup": "extensions",
      "apiVersion": "v1beta1"
    },
    "responseStatus": {
      "metadata": {},
      "code": 201
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "extensions/v1beta1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "creationTimestamp": null,
        "annotations": {
          "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"extensions/v1beta1\",\"kind\":\"NetworkPolicy\",\"metadata\":{\"annotations\":{},\"name\":\"test-audit-k8s-policy\",\"namespace\":\"default\"},\"spec\":{\"ingress\":[{\"from\":[{\"namespaceSelector\":{\"matchLabels\":{\"app\":\"connect-test\"}}}]}],\"podSelector\":{},\"policyTypes\":[\"Ingress\"]}}\n"
        }
      },
      "spec": {
        "podSelector": {},
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "extensions/v1beta1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "selfLink": "/apis/extensions/v1beta1/namespaces/default/networkpolicies/test-audit-k8s-policy",
        "uid": "32a80af0-6dce-11e9-af27-42010a8000a7",
        "resourceVersion": "525736",
        "generation": 1,
        "creationTimestamp": "2019-05-03T18:06:39Z",
        "annotations": {
          "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"extensions/v1beta1\",\"kind\":\"NetworkPolicy\",\"metadata\":{\"annotations\":{},\"name\":\"test-audit-k8s-policy\",\"namespace\":\"default\"},\"spec\":{\"ingress\":[{\"from\":[{\"namespaceSelector\":{\"matchLabels\":{\"app\":\"connect-test\"}}}]}],\"podSelector\":{},\"policyTypes\":[\"Ingress\"]}}\n"
        }
      },
      "spec": {
        "podSelector": {},
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "connect-test"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T18:06:39.206972Z",
    "stageTimestamp": "2019-05-03T18:06:39.214054Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": ""
    },
    "name": "test-audit-k8s-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T18:06:39.206Z"
    ],
    "stageTimestamp": [
      "2019-05-03T18:06:39.214Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T18:06:39.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T18:06:39.000Z"
    ],
    "timestamp": [
      "2019-05-03T18:06:39.000Z"
    ]
  },
  "highlight": {
    "objectRef.name": [
      "@kibana-highlighted-field@test@/kibana-highlighted-field@-@kibana-highlighted-field@audit@/kibana-highlighted-field@-@kibana-highlighted-field@k8s@/kibana-highlighted-field@-@kibana-highlighted-field@policy@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556906799000
  ]
}
```

### Update

```
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "H13hfmoBmcPPPMIsL51Y",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T18:07:23Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T18:07:23Z",
    "auditID": "058bf307-949d-4b47-8aae-bef85f280b2f",
    "stage": "ResponseComplete",
    "requestURI": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
    "verb": "update",
    "user": {
      "username": "jane",
      "uid": "1",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "uid": "32a80af0-6dce-11e9-af27-42010a8000a7",
      "apiGroup": "networking.k8s.io",
      "apiVersion": "v1",
      "resourceVersion": "525736"
    },
    "responseStatus": {
      "metadata": {},
      "code": 200
    },
    "requestObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "uid": "32a80af0-6dce-11e9-af27-42010a8000a7",
        "resourceVersion": "525736",
        "creationTimestamp": null
      },
      "spec": {
        "podSelector": {},
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "test-connect"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "responseObject": {
      "kind": "NetworkPolicy",
      "apiVersion": "networking.k8s.io/v1",
      "metadata": {
        "name": "test-audit-k8s-policy",
        "namespace": "default",
        "selfLink": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
        "uid": "32a80af0-6dce-11e9-af27-42010a8000a7",
        "resourceVersion": "525815",
        "generation": 2,
        "creationTimestamp": "2019-05-03T18:06:39Z"
      },
      "spec": {
        "podSelector": {},
        "ingress": [
          {
            "from": [
              {
                "namespaceSelector": {
                  "matchLabels": {
                    "app": "test-connect"
                  }
                }
              }
            ]
          }
        ],
        "policyTypes": [
          "Ingress"
        ]
      }
    },
    "requestReceivedTimestamp": "2019-05-03T18:07:23.908106Z",
    "stageTimestamp": "2019-05-03T18:07:23.910647Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": "RBAC: allowed by ClusterRoleBinding \"jane-tigera\" of ClusterRole \"network-admin\" to User \"jane\""
    },
    "name": "test-audit-k8s-policy"
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T18:07:23.908Z"
    ],
    "stageTimestamp": [
      "2019-05-03T18:07:23.910Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T18:07:23.000Z"
    ],
    "responseObject.metadata.creationTimestamp": [
      "2019-05-03T18:06:39.000Z"
    ],
    "timestamp": [
      "2019-05-03T18:07:23.000Z"
    ]
  },
  "highlight": {
    "objectRef.name": [
      "@kibana-highlighted-field@test@/kibana-highlighted-field@-@kibana-highlighted-field@audit@/kibana-highlighted-field@-@kibana-highlighted-field@k8s@/kibana-highlighted-field@-@kibana-highlighted-field@policy@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556906843000
  ]
}
```

### Delete
```
{
  "_index": "tigera_secure_ee_audit_kube.cluster.20190503",
  "_type": "fluentd",
  "_id": "KF3hfmoBmcPPPMIsR52Q",
  "_version": 1,
  "_score": null,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-03T18:07:31Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-03T18:07:31Z",
    "auditID": "07c861f3-f5cb-4a39-b71e-4976db2f9f76",
    "stage": "ResponseComplete",
    "requestURI": "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies/test-audit-k8s-policy",
    "verb": "delete",
    "user": {
      "username": "jane",
      "uid": "1",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.175"
    ],
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36",
    "objectRef": {
      "resource": "networkpolicies",
      "namespace": "default",
      "name": "test-audit-k8s-policy",
      "apiGroup": "networking.k8s.io",
      "apiVersion": "v1"
    },
    "responseStatus": {
      "metadata": {},
      "status": "Success",
      "code": 200
    },
    "responseObject": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Success",
      "details": {
        "name": "test-audit-k8s-policy",
        "group": "networking.k8s.io",
        "kind": "networkpolicies",
        "uid": "32a80af0-6dce-11e9-af27-42010a8000a7"
      }
    },
    "requestReceivedTimestamp": "2019-05-03T18:07:31.610998Z",
    "stageTimestamp": "2019-05-03T18:07:31.615144Z",
    "annotations": {
      "authorization.k8s.io/decision": "allow",
      "authorization.k8s.io/reason": "RBAC: allowed by ClusterRoleBinding \"jane-tigera\" of ClusterRole \"network-admin\" to User \"jane\""
    },
    "name": null
  },
  "fields": {
    "requestReceivedTimestamp": [
      "2019-05-03T18:07:31.610Z"
    ],
    "stageTimestamp": [
      "2019-05-03T18:07:31.615Z"
    ],
    "metadata.creationTimestamp": [
      "2019-05-03T18:07:31.000Z"
    ],
    "timestamp": [
      "2019-05-03T18:07:31.000Z"
    ]
  },
  "highlight": {
    "objectRef.name": [
      "@kibana-highlighted-field@test@/kibana-highlighted-field@-@kibana-highlighted-field@audit@/kibana-highlighted-field@-@kibana-highlighted-field@k8s@/kibana-highlighted-field@-@kibana-highlighted-field@policy@/kibana-highlighted-field@"
    ]
  },
  "sort": [
    1556906851000
  ]
}
```

## Status
```
{
  "_index": "tigera_secure_ee_audit_ee.cluster.20190507",
  "_type": "fluentd",
  "_id": "fgSXkGoBxB1mk6MND1lZ",
  "_score": 1.3862944,
  "_source": {
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1beta1",
    "metadata": {
      "creationTimestamp": "2019-05-07T04:39:34Z"
    },
    "level": "RequestResponse",
    "timestamp": "2019-05-07T04:39:34Z",
    "auditID": "62d4a78d-1067-42b6-9c70-a4aaf6cf2cbc",
    "stage": "ResponseComplete",
    "requestURI": "/apis/projectcalico.org/v3/globalnetworkpolicies/test.allow-egress-to-domains",
    "verb": "update",
    "user": {
      "username": "jane",
      "groups": [
        "system:authenticated"
      ]
    },
    "sourceIPs": [
      "10.128.0.12"
    ],
    "objectRef": {
      "resource": "globalnetworkpolicies",
      "name": "test.allow-egress-to-domains",
      "uid": "c9534716-7081-11e9-8d6c-42010a80007d",
      "apiGroup": "projectcalico.org",
      "apiVersion": "v3",
      "resourceVersion": "74616"
    },
    "responseStatus": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Failure",
      "message": " \"test.allow-egress-to-domains\" is invalid: Destination.Selector: Invalid value: \"null\": must be left empty when Destination.Domains is specified",
      "reason": "Invalid",
      "details": {
        "name": "test.allow-egress-to-domains",
        "causes": [
          {
            "reason": "FieldValueInvalid",
            "message": "Invalid value: \"null\": must be left empty when Destination.Domains is specified",
            "field": "Destination.Selector"
          }
        ]
      },
      "code": 422
    },
    "requestObject": {
      "kind": "GlobalNetworkPolicy",
      "apiVersion": "projectcalico.org/v3",
      "metadata": {
        "name": "test.allow-egress-to-domains",
        "uid": "c9534716-7081-11e9-8d6c-42010a80007d",
        "resourceVersion": "74616",
        "creationTimestamp": null
      },
      "spec": {
        "tier": "test",
        "order": 100,
        "egress": [
          {
            "action": "Allow",
            "source": {},
            "destination": {
              "selector": "my-pod-label == \"my-value\"",
              "domains": [
                "api.alice.com",
                "bob.example.com"
              ]
            }
          }
        ],
        "selector": "my-pod-label == \"my-value\"",
        "types": [
          "Egress"
        ]
      }
    },
    "responseObject": {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {},
      "status": "Failure",
      "message": " \"test.allow-egress-to-domains\" is invalid: Destination.Selector: Invalid value: \"null\": must be left empty when Destination.Domains is specified",
      "reason": "Invalid",
      "details": {
        "name": "test.allow-egress-to-domains",
        "causes": [
          {
            "reason": "FieldValueInvalid",
            "message": "Invalid value: \"null\": must be left empty when Destination.Domains is specified",
            "field": "Destination.Selector"
          }
        ]
      },
      "code": 422
    },
    "requestReceivedTimestamp": "2019-05-07T04:39:34.945734Z",
    "stageTimestamp": "2019-05-07T04:39:34.953272Z",
    "name": null
  }
}
```
