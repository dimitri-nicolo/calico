module github.com/tigera/calicoq

go 1.13

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/go-ini/ini v1.49.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v2.6.0-0.dev+incompatible
)

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200318014531-3f44cbe94f88
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200319165815-dcfd07befeb2
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.4.2-0.20190403091019-9b3cdde74fbe
)
