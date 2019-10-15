module github.com/kelseyhightower/confd

go 1.12

require (
	cloud.google.com/go v0.38.0
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.2.2
	github.com/Masterminds/sprig v2.19.0+incompatible
	github.com/PuerkitoBio/purell v1.1.1
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/aokoli/goutils v1.1.0
	github.com/aws/aws-sdk-go v1.23.22 // indirect
	github.com/beorn7/perks v1.0.0
	github.com/coreos/etcd v3.3.15+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/jsonpointer v0.19.2
	github.com/go-openapi/jsonreference v0.19.2
	github.com/go-openapi/spec v0.19.2
	github.com/go-openapi/swag v0.19.2
	github.com/go-playground/locales v0.12.1
	github.com/go-playground/universal-translator v0.16.1-0.20170327191703-71201497bace
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.1
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c
	github.com/google/gofuzz v1.0.0
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.0.0-20170729233727-0c5108395e2d
	github.com/gophercloud/gophercloud v0.1.0
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7
	github.com/hashicorp/consul/api v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1
	github.com/hashicorp/vault/api v1.0.4 // indirect
	github.com/huandu/xstrings v1.2.0
	github.com/imdario/mergo v0.3.5
	github.com/json-iterator/go v1.1.7
	github.com/kelseyhightower/envconfig v1.3.1-0.20180517194557-dd1402a4d99d
	github.com/kelseyhightower/memkv v0.1.1
	github.com/leodido/go-urn v1.1.1-0.20181204092800-a67a23e1c1af
	github.com/mailru/easyjson v0.0.0-20190626092158-b2ccc519800e
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pborman/uuid v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/projectcalico/confd v3.2.0+incompatible // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/typha v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v0.9.4
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.4.1
	github.com/prometheus/procfs v0.0.2
	github.com/robfig/cron v1.2.0
	github.com/samuel/go-zookeeper v0.0.0-20190810000440-0ceca61e4d75 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.3
	github.com/ugorji/go v1.1.7 // indirect
	github.com/xordataexchange/crypt v0.0.2 // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	google.golang.org/appengine v1.5.0
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873
	google.golang.org/grpc v1.23.0
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/inf.v0 v0.9.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20191010143144-fbf594f18f80
	k8s.io/apimachinery v0.0.0-20191014065749-fb3eea214746
	k8s.io/apiserver v0.0.0-20191010200905-b803d9d0d3eb
	k8s.io/client-go v0.0.0-20191010200049-172b42569cca
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191013032835-a13baff29e71
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20191015080800-8447f77e8aa6
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
