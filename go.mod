module github.com/projectcalico/cni-plugin

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/Microsoft/hcsshim v0.0.0-20190402014724-5f3c4ba7af30
	github.com/alexflint/go-filemutex v0.0.0-20171022225611-72bdc8eae2ae // indirect
	github.com/buger/jsonparser v0.0.0-20180808090653-f4dd9f5a6b44
	github.com/containernetworking/cni v0.0.0-20180705210735-e67bb289cccf
	github.com/containernetworking/plugins v0.0.0-20180925020009-646dbbace1b1
	github.com/coreos/go-iptables v0.3.0 // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/juju/clock v0.0.0-20180808021310-bab88fc67299
	github.com/juju/errors v0.0.0-20180806074554-22422dad46e1
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/juju/retry v0.0.0-20180821225755-9058e192b216 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/juju/version v0.0.0-20180108022336-b64dbd566305 // indirect
	github.com/mcuadros/go-version v0.0.0-20190308113854-92cdf37c5b75
	github.com/natefinch/atomic v0.0.0-20150920032501-a62ce929ffcc
	github.com/onsi/ginkgo v0.0.0-20170829012221-11459a886d9c
	github.com/onsi/gomega v0.0.0-20170829124025-dcabb60a477c
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/rakelkar/gonetsh v0.0.0-20180118190048-758b1f7c9d1c
	github.com/safchain/ethtool v0.0.0-20170622225139-7ff1ba29eca2 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.2.0
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vishvananda/netlink v0.0.0-20170630184320-6e453822d85e
	github.com/vishvananda/netns v0.0.0-20170219233438-54f0e4339ce7 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/utils v0.0.0-20180918230422-cd34563cd63c
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191120194820-00382998ff0b
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
