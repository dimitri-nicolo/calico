module github.com/tigera/es-proxy

go 1.11

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v2.4.1-0.20190605125757-c00322e56455+incompatible
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v2.6.0-0.dev.0.20190718111044-9bef69d2b882+incompatible // indirect
)

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/aquasecurity/kube-bench v0.0.29 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190515213511-eb9f6a1743f3 // indirect
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96 // indirect
	github.com/elazarl/goproxy v0.0.0-20170405201442-c4fc26588b6e // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/evanphx/json-patch v4.2.0+incompatible // indirect
	github.com/fatih/color v1.5.0 // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-openapi/spec v0.19.0 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/golang/groupcache v0.0.0-20160516000752-02826c3e7903 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/hcl v0.0.0-20171017181929-23c074d0eceb // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.8-0.20190531063913-f757d8626a73 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jinzhu/gorm v0.0.0-20160404144928-5174cc5c242a // indirect
	github.com/jinzhu/inflection v0.0.0-20170102125226-1c35d901db3d // indirect
	github.com/jinzhu/now v1.0.1 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lib/pq v0.0.0-20171126050459-83612a56d3dd // indirect
	github.com/magiconair/properties v0.0.0-20171031211101-49d762b9817b // indirect
	github.com/mattn/go-colorable v0.0.0-20170210172801-5411d3eea597 // indirect
	github.com/mattn/go-isatty v0.0.0-20170307163044-57fdcb988a5c // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mitchellh/mapstructure v0.0.0-20171017171808-06020f85339e // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/olivere/elastic v6.2.21+incompatible // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pelletier/go-toml v0.0.0-20171222114548-0131db6d737c // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/projectcalico/felix v3.7.3+incompatible // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee // indirect
	github.com/projectcalico/libcalico-go v3.9.0-0.dev.0.20190719174102-241d8b0486a3+incompatible
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/spf13/afero v0.0.0-20171228125011-57afd63c6860 // indirect
	github.com/spf13/cast v1.1.0 // indirect
	github.com/spf13/cobra v0.0.1 // indirect
	github.com/spf13/jwalterweatherman v0.0.0-20170901151539-12bd96e66386 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/spf13/viper v1.0.0 // indirect
	github.com/tigera/calico-k8sapiserver v2.4.0-0.dev.0.20190627044251-48f513106b84+incompatible // indirect
	github.com/tigera/compliance v2.6.0-0.dev.0.20190719220854-7f8974735d8c+incompatible
	github.com/tigera/licensing v2.2.3+incompatible // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980 // indirect
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.21.1 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20190308202827-072894a440bd
	k8s.io/apimachinery v0.0.0-20190308202827-103fd098999d
	k8s.io/apiserver v0.0.0-20180327025904-5ae41ac86efd // indirect
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/klog v0.3.3 // indirect
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
