module github.com/projectcalico/libcalico-go

go 1.15

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v0.0.0-20180807142431-c84ddcca87bf // indirect
	github.com/Masterminds/sprig v2.19.0+incompatible
	github.com/alecthomas/participle v0.3.0
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/go-openapi/spec v0.19.3
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace // indirect
	github.com/google/gopacket v1.1.17
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.11.1 // indirect
	github.com/huandu/xstrings v0.0.0-20180906151751-8bbcf2f9ccb5 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.0
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	gonum.org/v1/netlib v0.0.0-20190331212654-76723241ea4e // indirect
	google.golang.org/grpc v1.27.0
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect

	// validator.v9 must be at v9.30.2
	gopkg.in/go-playground/validator.v9 v9.30.2
	gopkg.in/tchap/go-patricia.v2 v2.2.6
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/apiserver v0.19.6
	k8s.io/client-go v0.19.6
	k8s.io/code-generator v0.19.6
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
)

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
