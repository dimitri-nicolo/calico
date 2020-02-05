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
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/typha v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v0.0.0-20200103202009-cd8a8b68f3ac // indirect

	// k8s.io/api v1.16.3 is at 16d7abae0d2a
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200203232830-a5de53a78f58
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200204210600-5a782d0a079c
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
