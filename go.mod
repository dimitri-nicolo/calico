module github.com/kelseyhightower/confd

go 1.15

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/go-playground/universal-translator v0.16.1-0.20170327191703-71201497bace // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/kelseyhightower/memkv v0.1.1
	github.com/leodido/go-urn v1.1.1-0.20181204092800-a67a23e1c1af // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.2-0.20210518153130-35bb3f8c906d
	github.com/projectcalico/typha v0.7.3-0.20210518155147-75c42c949f88
	github.com/sirupsen/logrus v1.7.0
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210511163730-078ad1161770
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210513005647-366239b8ed7a
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
