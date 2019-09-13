module github.com/projectcalico/libcalico-go

go 1.12

require (
	github.com/Azure/go-autorest v10.6.2+incompatible // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v0.0.0-20180807142431-c84ddcca87bf // indirect
	github.com/Masterminds/sprig v2.19.0+incompatible
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.8+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v0.0.0-20160705203006-01aeca54ebda // indirect
	github.com/go-openapi/runtime v0.0.0-20180628220156-9a3091f566c0 // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace // indirect
	github.com/google/btree v0.0.0-20160524151835-7d79101e329e // indirect
	github.com/google/uuid v0.0.0-20171113160352-8c31c18f31ed // indirect
	github.com/gophercloud/gophercloud v0.0.0-20180330165814-781450b3c4fc // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.11.1 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20160207214719-a0d98a5f2880 // indirect
	github.com/huandu/xstrings v0.0.0-20180906151751-8bbcf2f9ccb5 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af // indirect
	github.com/onsi/ginkgo v0.0.0-20170829012221-11459a886d9c
	github.com/onsi/gomega v0.0.0-20170829124025-dcabb60a477c
	github.com/pborman/uuid v0.0.0-20150603214016-ca53cad383ca // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/common v0.2.0 // indirect
	github.com/robfig/cron v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.2.0
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v0.0.0-20171019201919-bdcc60b419d1 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/time v0.0.0-20161028155119-f51c12702a4d // indirect
	google.golang.org/appengine v1.5.0 // indirect
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.27.0
	gopkg.in/inf.v0 v0.9.0 // indirect
	gopkg.in/tchap/go-patricia.v2 v2.3.0
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190324105220-f881eae9ec04
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190831074504-732c9ca86353
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
)

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
