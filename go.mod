module github.com/tigera/ingress-collector

go 1.18

require (
	github.com/gogo/protobuf v1.3.2
	github.com/hpcloud/tail v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.1
	github.com/projectcalico/calico v3.21.2+incompatible
	github.com/sirupsen/logrus v1.9.0
	golang.org/x/net v0.3.1-0.20221206200815-1e63c2f08a10
	google.golang.org/grpc v1.50.1
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/jinzhu/copier v0.3.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/tigera/api v0.0.0-20211211005256-a3f1397dfea5 // indirect
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/term v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	golang.org/x/time v0.0.0-20220609170525-579cf78fd858 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221024183307-1bc688fe9f3e // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.25.6 // indirect
	k8s.io/apimachinery v0.25.6 // indirect
	k8s.io/apiserver v0.25.6 // indirect
	k8s.io/client-go v0.25.6 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	// Replace the envoy data-plane-api dependency with the projectcalico fork that includes the generated
	// go bindings for the API. Upstream only includes the protobuf definitions, so we need to fork in order to
	// supply the go code.
	github.com/envoyproxy/data-plane-api => github.com/projectcalico/data-plane-api v0.0.0-20210121211707-a620ff3c8f7e
	github.com/projectcalico/calico => github.com/tigera/calico-private v1.11.0-cni-plugin-private.0.20230128203806-a88481d1b98d
	github.com/tigera/api => github.com/tigera/calico-private/api v0.0.0-20230128203806-a88481d1b98d

	k8s.io/api => k8s.io/api v0.25.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.6
	k8s.io/apiserver => k8s.io/apiserver v0.25.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.25.6
	k8s.io/client-go => k8s.io/client-go v0.25.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.25.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.25.6
	k8s.io/code-generator => k8s.io/code-generator v0.25.6
	k8s.io/component-base => k8s.io/component-base v0.25.6
	k8s.io/component-helpers => k8s.io/component-helpers v0.25.6
	k8s.io/controller-manager => k8s.io/controller-manager v0.25.6
	k8s.io/cri-api => k8s.io/cri-api v0.25.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.25.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.25.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.25.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.6
	k8s.io/kubectl => k8s.io/kubectl v0.25.6
	k8s.io/kubelet => k8s.io/kubelet v0.25.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.25.6
	k8s.io/metrics => k8s.io/metrics v0.25.6
	k8s.io/mount-utils => k8s.io/mount-utils v0.25.6
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.25.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.25.6
)
