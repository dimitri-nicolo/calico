module github.com/projectcalico/kube-controllers

go 1.12

require (
	cloud.google.com/go v0.26.0
	github.com/Azure/go-autorest v10.6.2+incompatible
	github.com/Masterminds/semver v1.2.2
	github.com/Masterminds/sprig v2.19.0+incompatible
	github.com/PuerkitoBio/purell v1.1.1
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/aokoli/goutils v1.1.0
	github.com/aws/aws-sdk-go v1.13.54
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.0.0+incompatible
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v0.0.0-20190327024845-3be5ad479f69
	github.com/go-openapi/jsonpointer v0.19.2
	github.com/go-openapi/jsonreference v0.19.2
	github.com/go-openapi/spec v0.19.2
	github.com/go-openapi/swag v0.19.2
	github.com/go-playground/locales v0.12.1
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/btree v1.0.0
	github.com/google/gofuzz v1.0.0
	github.com/google/gopacket v0.0.0-20190313190028-b7586607157b
	github.com/google/uuid v0.0.0-20171113160352-8c31c18f31ed
	github.com/googleapis/gnostic v0.3.1
	github.com/gophercloud/gophercloud v0.4.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/hpcloud/tail v1.0.0
	github.com/huandu/xstrings v1.2.0
	github.com/imdario/mergo v0.3.7
	github.com/jmespath/go-jmespath v0.0.0-20151117175822-3433f3ea46d9
	github.com/joho/godotenv v1.3.0
	github.com/json-iterator/go v1.1.7
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af
	github.com/mailru/easyjson v0.0.0-20190614124828-94de47d64c63
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627
	github.com/pborman/uuid v0.0.0-20150603214016-ca53cad383ca
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/projectcalico/felix v0.0.0-00010101000000-000000000000
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/typha v3.8.2+incompatible
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/prometheus/common v0.2.0
	github.com/prometheus/procfs v0.0.0-20181204211112-1dc9a6cbc91a
	github.com/robfig/cron v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.3
	github.com/tigera/licensing v2.6.0-0.dev+incompatible
	github.com/tigera/nfnetlink v0.0.0-20190401090543-2623d65568be
	github.com/vishvananda/netns v0.0.0-20170219233438-54f0e4339ce7
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	google.golang.org/appengine v1.5.0
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/grpc v1.19.0
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/square/go-jose.v2 v2.0.0-20180411045311-89060dee6a84
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190324105220-f881eae9ec04
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/klog v0.4.0
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/utils v0.0.0-20190221042446-c2654d5206da
)

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20190930195253-274e777a2f69
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191001180844-80bdc7c2bee3
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20191001104204-33e4233e9f3f
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
