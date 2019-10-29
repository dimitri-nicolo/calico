module github.com/kelseyhightower/confd

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.2.2 // indirect
	github.com/go-playground/universal-translator v0.16.1-0.20170327191703-71201497bace // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.3.1-0.20180517194557-dd1402a4d99d // indirect
	github.com/kelseyhightower/memkv v0.1.1
	github.com/leodido/go-urn v1.1.1-0.20181204092800-a67a23e1c1af // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/projectcalico/typha v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191029214447-cf245bbeba6d
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20191023103646-6c51073f7cfc
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
