module github.com/rafaelvanoni/felix-private

go 1.12

require (
	cloud.google.com/go v0.0.0-20160913182117-3b1ae45394a2
	github.com/Azure/go-autorest v10.6.2+incompatible
	github.com/Masterminds/semver v1.2.2
	github.com/Masterminds/sprig v2.17.1+incompatible
	github.com/Microsoft/go-winio v0.0.0-20190408173621-84b4ab48a507
	github.com/Microsoft/hcsshim v0.0.0-20190408221605-063ae4a83d78
	github.com/PuerkitoBio/purell v1.1.0
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/aokoli/goutils v1.1.0
	github.com/aws/aws-sdk-go v1.13.54
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/containernetworking/cni v0.5.2
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v0.0.0-20160705203006-01aeca54ebda
	github.com/docopt/docopt-go v0.0.0-20160216232012-784ddc588536
	github.com/emicklei/go-restful v0.0.0-20170410110728-ff4f55a20633
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/ghodss/yaml v0.0.0-20150909031657-73d445a93680
	github.com/go-ini/ini v0.0.0-20190327024845-3be5ad479f69
	github.com/go-openapi/jsonpointer v0.19.0
	github.com/go-openapi/jsonreference v0.19.0
	github.com/go-openapi/spec v0.0.0-20180801175345-384415f06ee2
	github.com/go-openapi/swag v0.17.0
	github.com/go-playground/locales v0.12.1
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace
	github.com/gobuffalo/packr v0.0.0-20190404150745-68a5fdc58d98
	github.com/gogo/protobuf v1.1.1
	github.com/golang/glog v0.0.0-20141105023935-44145f04b68c
	github.com/golang/protobuf v1.2.0
	github.com/google/btree v0.0.0-20160524151835-7d79101e329e
	github.com/google/gofuzz v0.0.0-20161122191042-44d81051d367
	github.com/google/gopacket v0.0.0-20190313190028-b7586607157b
	github.com/google/uuid v0.0.0-20171113160352-8c31c18f31ed
	github.com/googleapis/gnostic v0.0.0-20170729233727-0c5108395e2d
	github.com/gophercloud/gophercloud v0.0.0-20180330165814-781450b3c4fc
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7
	github.com/gxed/GoEndian v0.0.0-20160916112711-0f5c6873267e
	github.com/gxed/eventfd v0.0.0-20160916113412-80a92cca79a8
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.0.0-20160207214719-a0d98a5f2880
	github.com/hpcloud/tail v1.0.0
	github.com/huandu/xstrings v1.2.0
	github.com/imdario/mergo v0.0.0-20141206190957-6633656539c1
	github.com/ipfs/go-log v0.0.0-20180611222144-5dc2060baaf8
	github.com/jbenet/go-reuseport v0.0.0-20181102234943-8cfd5f2973c8
	github.com/jbenet/go-sockaddr v0.0.0-20160322130902-2e7ea655c10e
	github.com/jmespath/go-jmespath v0.0.0-20151117175822-3433f3ea46d9
	github.com/json-iterator/go v0.0.0-20180612202835-f2b4162afba3
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af
	github.com/libp2p/go-reuseport v0.0.0-20180924121034-dd0c37d7767b
	github.com/libp2p/go-sockaddr v0.0.0-20190411201116-52957a0228cc
	github.com/mailru/easyjson v0.0.0-20180823135443-60711f1a8329
	github.com/mattn/go-colorable v0.1.1
	github.com/mattn/go-isatty v0.0.7
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v0.0.0-20180320133207-05fbef0ca5da
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pborman/uuid v0.0.0-20150603214016-ca53cad383ca
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/projectcalico/felix v3.8.2+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v3.7.5+incompatible
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/prometheus/common v0.2.0
	github.com/prometheus/procfs v0.0.0-20181005140218-185b4288413d
	github.com/robfig/cron v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/tigera/licensing v2.6.0-0.dev+incompatible
	github.com/tigera/nfnetlink v0.0.0-20190401090543-2623d65568be
	github.com/vishvananda/netns v0.0.0-20160430053723-8ba1072b58e0
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	golang.org/x/oauth2 v0.0.0-20170412232759-a6bd8cefa181
	golang.org/x/sys v0.0.0-20190228124157-a34e9553db1e
	golang.org/x/text v0.3.0
	golang.org/x/time v0.0.0-20161028155119-f51c12702a4d
	google.golang.org/appengine v1.5.0
	google.golang.org/genproto v0.0.0-20170731182057-09f6ed296fc6
	google.golang.org/grpc v1.7.5
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/inf.v0 v0.9.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/square/go-jose.v2 v2.0.0-20180411045311-89060dee6a84
	gopkg.in/tchap/go-patricia.v2 v2.3.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190324105220-f881eae9ec04
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20180620173706-91cfa479c814
)

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
