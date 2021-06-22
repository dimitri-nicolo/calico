# PacketCapture Download API

PacketCapture Download API exposes the API to retrieve pcap files generated by a PacketCapture. This enables personas
like DevOps or Platform Engineers to retrieve files backed by a namespace RBAC.  

Sample call for management or standalone cluster:
```
kubectl port-forward -ntigera-manager service/tigera-manager 9443:9443 &
NS=<REPLACE_WITH_PACKETCAPTURE_NS> NAME=<REPLACE_WITH_PACKETCAPTURE_NAME> TOKEN=<REPLACE_WITH_YOUR_TOKEN> \
curl "https://localhost:9443/packet-capture/download/$NS/$NAME/files.zip" \
-H "Authorizaton: Bearer $TOKEN" \
-O -L
```

Files from a managed cluster can be retrieved via the management cluster:

```
kubectl port-forward -ntigera-manager service/tigera-manager 9443:9443 &
NS=<REPLACE_WITH_PACKETCAPTURE_NS> NAME=<REPLACE_WITH_PACKETCAPTURE_NAME> TOKEN=<REPLACE_WITH_YOUR_TOKEN> CLUSTER=<REPLACE_WITH_CLUSTER> \
curl 'https://localhost:9443/packet-capture/download/$NS/$NAME/files.zip \
-H "Authorizaton: Bearer $TOKEN" \
-H "X-CLUSTER-ID: $CLUSTER" \
-O -L
```

A request like above will retrieve all the generated pcap files at the moment of download as a zip archive.

In order to allow access to this API, a role like the following needs to be applied in a K8s cluster:

```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: capture-files-role
  namespace: dev
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["packetcaptures/files"]
  resourceNames: ["*"]
  verbs: ["get"]
```

## Calico Enterprise

PacketCapture API is a service that is deployed as container part of tigera-manager deployment. By default, it is
installed by Tigera operator. This service will only be deployed only for Standalone and Management clusters, and requests
will be routed to the appropriate managed clusters.

Voltron container has route defined for any requests that starts with `/packet-capture/` prefix. It will strip away that 
prefix and forward it to the container where this service resides.

## Building and testing

In order to build the API locally, use one of the following commands:

```
make image
```

or

```
make ci cd
```

Testing this API on a local environment is currently not possible as it has dependency to run in an K8S environment
with Calico Enterprise installed.

In order to run local tests, use the following command:

```
make ut
```

## Configuration and permissions

| ENV        | Default value          | Description  |
| ------------- |:-------------:| -----:|
| PACKETCAPTURE_API_PORT      | `8444` | Local Port to start the service |
| PACKETCAPTURE_API_HOST      | <empty>      |   Host for the service |
| PACKETCAPTURE_API_LOG_LEVEL | `Info`      |    Log Level across service |
| PACKETCAPTURE_API_DEX_ENABLED | `False`      |    Enable Dex for authentication |
| PACKETCAPTURE_API_DEX_ISSUER | `https://127.0.0.1:5556/dex`      |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_CLIENT_ID | `tigera-manager`      |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_JWK_URL | `https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys`     |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_USER_CLAIM | `email`      |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_GROUPS_CLAIM | <empty>      |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_USERNAME_PREFIX | <empty>      |    Dex Setup Configuration |
| PACKETCAPTURE_API_DEX_GROUPS_PREFIX | <empty>      |    Dex Setup Configuration |
| PACKETCAPTURE_API_MULTI_CLUSTER_FORWARDING_CA | `/manager-tls/cert`      |    CA certificate for multicluster communication |
| PACKETCAPTURE_API_MULTI_CLUSTER_FORWARDING_ENDPOINT | `https://localhost:9443`      |    CA endpoint for multicluster communication |


This API makes use of tigera-manager service account and requires the following permissions:
- GET for api group `projectcalico.org` for resource `packetcaptures`
- CREATE for api group `projectcalico.org` for resource `authenticationreviews`
- LIST,WATCH,GET for api group `projectcalico.org` for resource `managedclusters`
- LIST for core v1 group for resource `pods` in namespace tigera-fluentd
- CREATE for core v1 group for subresource `pods/exec` in namespace tigera-fluentd
- CREATE for api group `authorization.k8s.io` for resource `subjectaccessreviews`


## Docs
- [How To - Packet Capture](https://docs.tigera.io/visibility/packetcapture)
- [PacketCapture Resource definition](https://docs.tigera.io/reference/resources/packetcapture)
- [Technical spec - V1](https://docs.google.com/document/d/1gsiogi9kdXDTFjtIOiXoI2uFk47kVZclqc0Gqqs8fLg/edit?usp=sharing)  
- [Technical spec - V2](https://docs.google.com/document/d/14suOeADcIcH8pmG64VjFiCjC44J3JQYRZTfXr6cpfEc/edit?usp=sharing)
- [3.3 TOI](https://docs.google.com/presentation/d/1LRYT9Aqm8ak5crg0UGGIQwui7LgcUTXhI8AK-dO1E7U/edit?usp=sharing)
