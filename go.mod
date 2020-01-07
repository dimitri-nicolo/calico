module tigera/calicoq

go 1.12

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/go-ini/ini v1.49.0 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/calicoq v2.6.1+incompatible
	github.com/tigera/licensing v2.6.0-0.dev+incompatible
	gopkg.in/square/go-jose.v2 v2.1.3 // indirect
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d // indirect
)

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20191220191724-c757233f7c16
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191216182023-9c9d4fc21fbd
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.4.2-0.20190403091019-9b3cdde74fbe
)
