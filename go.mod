module github.com/projectcalico/calicoctl

go 1.12

require (
	github.com/StackExchange/wmi v0.0.0-20181212234831-e0a55b97c705 // indirect
	github.com/coreos/etcd v3.3.10+incompatible // indirect
	github.com/docopt/docopt-go v0.0.0-20160216232012-784ddc588536
	github.com/eapache/channels v1.1.0 // indirect
	github.com/eapache/queue v0.0.0-20180227141424-093482f3f8ce // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/influxdata/influxdb v0.0.0-20190102202943-dd481f35df2c // indirect
	github.com/influxdata/platform v0.0.0-20190117200541-d500d3cf5589 // indirect
	github.com/mcuadros/go-version v0.0.0-20190308113854-92cdf37c5b75
	github.com/olekukonko/tablewriter v0.0.0-20190409134802-7e037d187b0c
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/osrg/gobgp v0.0.0-20170802061517-bbd1d99396fe
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.2-0.20191015222346-3d38c58337f2
	github.com/shirou/gopsutil v0.0.0-20190323131628-2cbc9195c892
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/tigera/licensing v2.7.0-0.dev.0.20191114203016-3c126d3f9ffe+incompatible
	github.com/vishvananda/netlink v0.0.0-20180501223456-f07d9d5231b9 // indirect
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc // indirect
	gopkg.in/square/go-jose.v2 v2.0.0-20190111193340-cbf0fd6a984a
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191204211106-7ffd5430c66d
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
