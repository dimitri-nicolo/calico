module github.com/kelseyhightower/confd

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1

	github.com/Masterminds/semver v1.2.2 // indirect
	github.com/go-playground/universal-translator v0.16.1-0.20170327191703-71201497bace // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/memkv v0.1.1
	github.com/leodido/go-urn v1.1.1-0.20181204092800-a67a23e1c1af // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/projectcalico/typha v0.7.3-0.20200601160722-b55a2f1ba49b
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v1.0.1-0.20200618213636-f5bf5cf12edb // indirect
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200715200940-967f2c6d1fd7
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20200715220132-e39d468d61f3
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
