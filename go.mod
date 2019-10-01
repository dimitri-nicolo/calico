module github.com/projectcalico/typha

go 1.12

require (
	github.com/Masterminds/semver v1.2.2 // indirect
	github.com/Workiva/go-datastructures v1.0.50
	github.com/docopt/docopt-go v0.0.0-20160216232012-784ddc588536
	github.com/go-ini/ini v0.0.0-20190327024845-3be5ad479f69
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/onsi/ginkgo v0.0.0-20170829012221-11459a886d9c
	github.com/onsi/gomega v0.0.0-20170829124025-dcabb60a477c
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pquerna/ffjson v0.0.0-20190813045741-dac163c6c0a9 // indirect
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v0.9.2
	github.com/sirupsen/logrus v1.2.0
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	gopkg.in/go-playground/validator.v9 v9.28.0 // indirect
	gopkg.in/ini.v1 v1.46.0 // indirect
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20190927100432-f0e4e03dabf5

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
