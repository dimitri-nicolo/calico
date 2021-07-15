module github.com/projectcalico/libcalico-go

go 1.15

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/alecthomas/participle v0.7.1
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/go-openapi/spec v0.19.5
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace // indirect
	github.com/google/gopacket v1.1.17
	github.com/google/uuid v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.6 // indirect
	github.com/huandu/xstrings v0.0.0-20180906151751-8bbcf2f9ccb5 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.0
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af // indirect
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/tchap/go-patricia/v2 v2.3.1
	github.com/tigera/api v0.0.0-20210714180252-70c5133a3886
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	google.golang.org/grpc v1.27.1
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect

	// validator.v9 must be at v9.30.2
	gopkg.in/go-playground/validator.v9 v9.30.2
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/code-generator v0.21.2
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/kind v0.11.1
)

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
