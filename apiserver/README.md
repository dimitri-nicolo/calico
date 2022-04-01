[![Build Status](https://semaphoreci.com/api/v1/projects/cdb95b8c-1d8c-4870-a15b-ba61c2eca39d/2613789/badge.svg)](https://semaphoreci.com/calico/apiserver)

# Tigera API server

k8s styled API server to interact with Calico resources.

To deploy, bring up k8s version v1.8+, preferably v1.9.

## Sample installation steps with kubeadm with Calico in ETCD mode

`sudo make kubeadm` automates steps 1-7 of this, if kubeadm is installed and the
docker image (`make tigera/cnx-apiserver`) has been built.
```
1. kubeadm reset; rm -rf /var/etcd
2. kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs. Refer artifacts/misc/kubeadm.yaml
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4. kubectl apply -f artifacts/misc/calico.yaml (this one has calico bringing up etcd 3.x backend)
5. kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7. kubectl create -f artifacts/example/ <-- The set of manifests necessary to install Aggregated API Server
   Prior to this, make sure you have checked out calico-k8sapiserver and have run make clean;make
   This will create the docker image needed by the example/rc.yaml
   OR docker tar can be found in: https://drive.google.com/open?id=0B1QYlddBYM-ZWkoxVWNfcFJtbUU
   docker load -i calico-k8sapiserver-latest.tar.xz
8. kubectl create -f artifacts/policies/policy.yaml <-- Creating a NetworkPolicy
9. kubectl create -f artifacts/policies/tier.yaml <-- Creating a Tier
10. kubectl create -f artifacts/policies/globalpolicy.yaml <-- Creating a GlobalNetworkPolicy
11. kubectl create -f artifacts/calico/globalnetworkset.yaml <-- Creating a GlobalNetworkSet
12. kubectl create -f artifacts/calico/iwantcake5-license.yaml <-- Creating a licenseKey
.
.
.
```

## Sample installation steps with kubeadm with Calico in KDD mode
```
1. kubeadm reset; rm -rf /var/etcd
2. kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs. Refer artifacts/misc/kubeadm.yaml
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4a. kubectl apply -f artifacts/misc/rbac-kdd.yaml
4b. kubectl apply -f artifacts/misc/kdd_calico.yaml
5. kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7a. kubectl create -f artifacts/example/
7b. kubectl create -f artifacts/example_kdd
8. kubectl create -f artifacts/policies/policy.yaml <-- Creating a NetworkPolicy
9. kubectl create -f artifacts/policies/tier.yaml <-- Creating a Tier
10. kubectl create -f artifacts/policies/globalpolicy.yaml <-- Creating a GlobalNetworkPolicy
11. kubectl create -f artifacts/calico/globalnetworkset.yaml <-- Creating a GlobalNetworkSet
12. kubectl create -f artifacts/calico/iwantcake5-license.yaml <-- Creating a licenseKey
.
.
.
```

## Cleanup and Reset
```
1. kubectl delete -f ~/go/src/github.com/tigera/calico-k8sapiserver/artifacts/example/
2. kubectl delete -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
3. rm -rf /var/etcd/
4. Reload/Rebuild the new latest docker image for calico-k8sapiserver
5. kubectl apply -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
6. kubectl create -f ~/go/src/github.com/tigera/calico-k8sapiserver/artifacts/example/
```

## Cluster Access/Authentication

Kubernetes natively supports various Authentication strategies like:
Client Certificates
Bearer Tokens
Authenticating proxy
HTTP Basic Auth

An aggregated API Server will be able to delegate authentication, of any incoming request, to the master API Server/Aggregator.

The Authentication strategy being setup as part of the above demo installation process (through kubeadm) is based on Client Certificates/PKI.

In order to make curl requests against the Aggergated API, following steps can be followed:

1. Open /etc/kubernetes/admin.conf
The file contains the bits of information that kubectl uses in order to make authorized requests against the API Server.

The file container client-certficate and client-key in base64 encoded format.

Copy the value of client-certificate-data in a file , say client-crt.
Run `base64 decode client-crt > client.crt`

Copy the value of client-key-data in a file, say client-key
Run `base64 decode client-key > client.key`

The curl command expects the client certificate to be presented in PEM format.

Generate the PEM file using the command:
`cat client.crt client.key > client.includesprivatekey.pem`

OR

use the helper script artifacts/misc/admin_conf_parser.py to generate /var/tmp/client.includesprivatekey.pem and use it in the
argument to the curl.

2. Find the API Server Certificate Authority info. This is used to verify the certificate response coming in from the Server.

The CA can be found under /etc/kubernetes/pki/apiserver.crt

3. Run the curl command against a K8s resource:

`curl --cacert /etc/kubernetes/pki/apiserver.crt --cert-type pem --cert client.includesprivatekey.pem https://10.0.2.15:6443/api/v1/nodes`

The API Server address can be found from the above admin.conf file as well.

The API Server command/flags used for running can be found under /etc/kubernetes/manifest/

## API Examples
```
Follows native Kubernetes REST semantics.
Tiers - APIVersion: projectcalico.org/v3 Kind: Tier
1. Listing Tiers: https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers
2. Getting a Tier: https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers/net-sec
3. Posting a Tier: -XPOST -d @tier.yaml  -H "Content-type:application/yaml"  https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers

NetworkPolicies - APIVersion: projectcalico.org/v3 Kind: NetworkPolicy
4. Listing networkpolicies across namespaces: https://10.0.2.15:6443/apis/projectcalico.org/v3/networkpolicies
5. Listing networkpolicy from a given namespace (belonging to default tier): https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies
^ NOTE: NetworkPolicy list will also include Core NetworkPolicies. Core NetworkPolicy names will be prepended with "knp."
^ NOTE: When fieldSelector not specified it defaults to "default" on the server side.
6. Watching networkpolicies in the default namespace: https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?watch=true
7. Selecting networkpolicies in the default namespace belonging to "net-sec": https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?fieldSelector=spec.tier=net-sec
8. Select networkpolicies based on Tier and watch at the same time: https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?watch=true&fieldSelector=spec.tier=net-sec
9. Create networkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies

GlobalNetworkPolicies - APIVersion: projectcalico.org/v3 Kind: GlobalNetworkPolicy
10. Listing globalnetworkpolicies: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies
11. Watching globalnetworkpolicies belonging to default tier: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies?watch=true
12. Selecting globalnetworkpolicies belonging to Tier1: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies?fieldSelector=spec.tier=Tier1
13. Create globalnetworkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies

Core/K8s NetworkPolicies - APIVersion: networking.k8s.io/v1 Kind: NetworkPolicy
14. Create core networkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/networking.k8s.io/v1/networkpolicies
NOTE: Use above endpoint for CREATE, UPDATE and DELETE on core networkpolicies.

Listing Namespaces - APIVersion: v1 Kind: Namespace
15. List K8s Namespaces:https://10.0.2.15:6443/api/v1/namespaces

GlobalNetworkSets - APIVersion: projectcalico.org/v3 Kind: GlobalNetworkSet
16. Listing globalnetworksets: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworksets
17. Getting a globalnetworkset: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworksets/sample-global-network-set
18. Posting a globalnetworkset: -XPOST -d @globalnetworkset.yaml  -H "Content-type:application/yaml"  https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworksets
19. Watching a globalnetworkset: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworksets/sample-global-network-set?watch=true

GlobalNetworkPolicies with Application Layer rules
20. Create globalnetworkpolicies with application layer rule: -XPOST -d @app-policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies

LicenseKeys - APIVersion: projectcalico.org/v3 Kind: LicenseKeys
21. Listing licenseKey: https://10.0.2.15:6443/apis/projectcalico.org/v3/licensekeys
22. Getting a licenseKey: https://10.0.2.15:6443/apis/projectcalico.org/v3/licensekeys/default

```

## Testing
The integration tests/functional verification tests can be run via the `fv`/`fv-kdd` Makefile target,
e.g.:

    $ make fv

The unit tests can be run via `ut` Makefile target,
e.g.:

    $ make ut

All of the tests can be run via `test` Makefile target,
e.g.:

    $ make test

## Adding resources to cnx-apiserver

Add the new resource definitions to tigera/api and associated code in libcalico-go.  Ensure the
code auto-generation is run (make gen-files) in both repositories.

The overall approach is largely identical for both namespaced (e.g. network policy)
as well as non-namespaced (e.g. globalnetworkset) resources:

1. Add the resource type definitions to `tigera/api`. This is
  likely comprised of a List `struct` type and an individual resource type. For
  example:

  ```go
  // +genclient:nonNamespaced
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

  // LicenseKeyList is a list of LicenseKey objects.
  type LicenseKeyList struct {
  	metav1.TypeMeta
  	metav1.ListMeta

  	Items []LicenseKey
  }

  // +genclient
  // +genclient:nonNamespaced
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  // If your object has a status subresource:
  // +kubebuilder:subresource:status

  type LicenseKey struct {
  	metav1.TypeMeta
  	metav1.ObjectMeta

  	Spec calico.LicenseKeySpec
  	// If your object has a status subresource:
  	Status calico.LicenseKeyStatus
  }
  ```

  Pay particular attention the `genclient` metadata - the above example is for
  a non-namespaced resource.

2. Add the k8s-facing resource types to `pkg/apis/projectcalico/v3/types.go`. This
  will be similar to the types above, except the metadata fields indicate how to
  pack/unpack json data. The contents will essentially use the Calico v3 resource
  type. For example:

  ```go
  // +genclient:nonNamespaced
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

  // LicenseKeyList  is a list of license objects.
  type LicenseKeyList struct {
  	metav1.TypeMeta `json:",inline"`
  	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

  	Items []LicenseKey `json:"items" protobuf:"bytes,2,rep,name=items"`
  }

  // +genclient
  // +genclient:nonNamespaced
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  // If your object has a status subresource:
  // +kubebuilder:subresource:status

  type LicenseKey struct {
  	metav1.TypeMeta   `json:",inline"`
  	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

  	Spec calico.LicenseKeySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
  	// If your object has a status subresource:
  	Status calico.LicenseKeyStatus `json:"status,omitempty" protobyf:"bytes,3,opt,name=status"`
  }

  ```

3. Once you have these type declarations, add the resource and resource list types
  to `api/pkg/apis/projectcalico/v3/register.go`. For example:

  ```go
  ...
  &LicenseKey{},
  &LicenseKeyList{},
  ...
  ```

4. Register the field label conversion anonymous function in `addConversionFuncs()`
  in `pkg/apis/projectcalico/v3/conversion.go`. For example:

  ```go
	err = scheme.AddFieldLabelConversionFunc(schema.GroupVersionKind{"projectcalico.org", "v3", "LicenseKey"},
		func(label, value string) (string, string, error) {
			switch label {
			case "metadata.name":
				return label, value, nil
			default:
				return "", "", fmt.Errorf("field label not supported: %s", label)
			}
		},
	)
	if err != nil {
		return err
	}
  ```

5. Add backend storage and associated strategies for your resource types. A simple
  approach is to just copy the two files below into your own package (and
  associated directory), and then modify the types to point to your resource
  declarations from the first couple of steps. For example:

  * `pkg/registry/projectcalico/licensekey/storage.go`
  * `pkg/registry/projectcalico/licensekey/strategy.go`

  If your resource has a status subresource, use the following instead:

  * `pkg/registry/projectcalico/globalthreatfeed/storage.go`
  * `pkg/registry/projectcalico/globalthreatfeed/strategy.go`

6. Add a reference to your storage strategy created in the previous steps to
  `pkg/registry/projectcalico/rest/storage_calico.go`. This registers a callback
  for REST API calls for your resource type. For example:

  ```go
  import (
    ...
    calicolicensekey "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/licensekey"
    )

  ...

  licenseKeyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("licensekeys"))
	if err != nil {
		return nil, err
	}
	licenseKeysSetOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   licenseKeyRESTOptions,
			Capacity:      10,
			ObjectType:    calicolicensekey.EmptyObject(),
			ScopeStrategy: calicolicensekey.NewStrategy(scheme),
			NewListFunc:   calicolicensekey.NewList,
			GetAttrsFunc:  calicolicensekey.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  licenseKeyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
	)

  ```

  If your resource has a status subresource, also register RESTapi calls for your status subresource:

  ```go
    ...
    gThreatFeedStatusRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalthreatfeeds/status"))
    if err != nil {
        return nil, err
    }
  
    gThreatFeedStatusOpts := server.NewOptions(
        sharedGlobalThreatFeedEtcdOpts,
        calicostorage.Options{
            RESTOptions:  gThreatFeedStatusRESTOptions,
            LicenseCache: licenseCache,
        },
        p.StorageType,
        authorizer,
        []string{},
    )
  ```

  Update the storage map (also in `pkg/registry/projectcalico/rest/storage_calico.go`)
  for your resource key with the associated REST api type, for example:

  ```go
  ...
  storage["licensekeys"] = calicolicensekey.NewREST(scheme, *licenseKeysSetOpts)
  ...
  ```

  If your resource has a status subresource:

  ```go
  ...
  globalThreatFeedsStorage, globalThreatFeedsStatusStorage := calicogthreatfeed.NewREST(scheme, *gThreatFeedOpts)
  storage["globalthreatfeeds"] = globalThreatFeedsStorage
  storage["globalthreatfeeds/status"] = globalThreatFeedsStatusStorage
  ...
  ```

7. Create a factory function to create a resource storage implementation. Use
  `pkg/storage/calico/licenseKey_storage.go` as a model for your work - this is
  basically a copy/paste and then update the resource type declarations.
  
  If your resource has a status subresource, also use 
  `pkg/storage/calico/globalThreatFeedStatus_storage.go` as model to create 
  subresource storage implementation.

8. Define how the API is going to be used by defining its behaviour is `hasRestrictionsFn()`
If an API is restricted by a license, you need to see if the feature is defined in the [licensing library](https://github.com/tigera/licensing/blob/master/client/features/features.go). A sample of implementing restrictions can be found at `pkg/storage/calico/globalReport_storage.go`

```go
hasRestrictionsFn := func(obj resourceObject) bool {
    return !opts.LicenseMonitor.GetFeatureStatus(features.AlertManagement)
}
```

9. Add your factory function to `NewStorage()` function in
  `pkg/storage/calico/storage_interface.go`. For example:

  ```go
  func NewStorage(...) {

  ...

    case "projectcalico.org/licensekeys":
		  return NewLicenseKeyStorage(opts)
    // If there is status subresource
    case "projectcalico.org/globalthreatfeeds/status":
        return NewGlobalThreatFeedStatusStorage(opts)
  }
  ```

10. Register your conversion routines in `pkg/storage/calico/converter.go`. For example:

  ```
    convertToAAPI() {
      ...
      case *libcalicoapi.LicenseKey:
        lcgLicense := libcalicoObject.(*libcalicoapi.LicenseKey)
        aapiLicenseKey := &aapi.LicenseKey{}
        LicenseKeyConverter{}.convertToAAPI(lcgLicense, aapiLicenseKey)
        return aapiLicenseKey
    }

  ```

  Remember to copy the Status member if your resource has one.

11. Lastly, add a clientset test for functional verification tests to
  `test/integration/clientset_test.go`. Take a look at the `TestLicenseKeyClient()`
  and `testLicenseKeyClient()` functions as an example.

* In order to rebuild the container image:

  ```bash
  make tigera/cnx-apiserver

  # In order to test:
  docker tag tigera/cnx-apiserver:latest quay.io/bcreane/cnx-apiserver:latest
  docker push quay.io/bcreane/cnx-apiserver:latest

  # then update the cnx-etcd.yaml manifest to change the cnx-apiserver image to
  # point to your image, e.g.

  apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    name: cnx-apiserver

  ...

  image: quay.io/bcreane/cnx-apiserver:latest
  imagePullPolicy: Always

  ```

* Verify you can view, modify, and create your resource via  `kubectl`, for example:

```bash
  kubectl get LicenseKeys
  kubectl delete licensekey default
  kubectl apply -f artifacts/calico/iwantcake5-license.yaml
```

* Run `make static-checks` before creating a PR to make sure that all the changes can be goimport-ed.

* For an example pull request that contains all these changes (plus all the
  generated files as well), see: https://github.com/tigera/calico-private/pull/4214

  For one with a status subresource, see:
  https://github.com/tigera/calico-k8sapiserver/pull/104 and
  https://github.com/tigera/calico-k8sapiserver/pull/106
