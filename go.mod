module github.com/tigera/es-proxy

go 1.13

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20191029225535-3f90fdd6c6ca
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191113223902-f77f93773a83 // indirect
	k8s.io/api => k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190324105220-f881eae9ec04
	k8s.io/client-go => k8s.io/client-go v8.0.0+incompatible
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190831074504-732c9ca86353
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
)

require (
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/calico-k8sapiserver v2.6.0-0.dev.0.20191030050937-be2e3fd6c28a+incompatible
	github.com/tigera/compliance v0.0.0-20191119174315-65936d01c64c
	github.com/tigera/lma v0.0.0-20191106193819-e9738ab8ba44
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20191009075622-910e671eb668
	k8s.io/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
	k8s.io/client-go v11.0.0+incompatible
)
