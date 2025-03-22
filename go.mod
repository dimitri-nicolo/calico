module github.com/projectcalico/calico

go 1.23.7

require (
	dario.cat/mergo v1.0.1
	github.com/BurntSushi/toml v1.5.0
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.12.9
	github.com/PaloAltoNetworks/pango v0.10.2
	github.com/SermoDigital/jose v0.9.2-0.20161205224733-f6df55f235c2
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/alecthomas/participle v0.7.1
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aquasecurity/kube-bench v0.10.4
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/aws/aws-sdk-go v1.55.6
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.9
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.210.0
	github.com/aws/smithy-go v1.22.3
	github.com/bits-and-blooms/bitset v1.22.0
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f
	github.com/bronze1man/goStrongswanVici v0.0.0-20221114103242-3f6dc524986c
	github.com/buger/jsonparser v1.1.1
	github.com/caimeo/iniflags v0.0.0-20171110233946-ef4ae6c5cd79
	github.com/cnf/structhash v0.0.0-20250313080605-df4c6cc74a9a
	// pod2daemon/cisdriver build failure after upgrading to v1.10.0+
	github.com/container-storage-interface/spec v1.9.0
	github.com/containernetworking/cni v1.2.3
	github.com/containernetworking/plugins v1.6.2
	github.com/corazawaf/coraza-coreruleset/v4 v4.7.0
	github.com/corazawaf/coraza/v3 v3.3.2
	github.com/coreos/go-iptables v0.8.0
	github.com/coreos/go-oidc v2.3.0+incompatible
	github.com/coreos/go-semver v0.3.1
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/distribution/reference v0.6.0
	github.com/docker/distribution v2.8.3+incompatible
	github.com/docker/docker v28.0.1+incompatible
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/elastic/cloud-on-k8s/v2 v2.16.1
	github.com/elastic/go-elasticsearch/v7 v7.17.10
	github.com/elazarl/goproxy v1.7.2
	github.com/elazarl/goproxy/ext v0.0.0-20250305112401-088f758167d2
	github.com/envoyproxy/go-control-plane/envoy v1.32.4
	github.com/felixge/httpsnoop v1.0.4
	github.com/florianl/go-nfqueue v1.3.2
	github.com/fluent/fluent-bit-go v0.0.0-20230731091245-a7a013e2473c
	github.com/fsnotify/fsnotify v1.8.0
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-ini/ini v1.67.0
	github.com/go-jose/go-jose/v4 v4.0.5
	github.com/go-kit/log v0.2.1
	github.com/go-logr/logr v1.4.2
	github.com/go-openapi/runtime v0.28.0
	github.com/go-sql-driver/mysql v1.9.0
	github.com/gofrs/flock v0.12.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/golang/protobuf v1.5.4
	github.com/golang/snappy v1.0.0
	github.com/google/btree v1.1.3
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v53 v53.2.0
	github.com/google/gopacket v1.1.19
	github.com/google/netstack v0.0.0-20191123085552-55fcc16cd0eb
	github.com/google/safetext v0.0.0-20240722112252-5a72de7e7962
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gruntwork-io/terratest v0.48.2
	github.com/hashicorp/yamux v0.1.2
	github.com/ishidawataru/sctp v0.0.0-20250303034628-ecf9ed6df987
	github.com/jarcoal/httpmock v1.3.1
	github.com/jcchavezs/mergefs v0.1.0
	github.com/joho/godotenv v1.5.1
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.12
	github.com/juju/clock v1.1.1
	github.com/juju/errors v1.0.0
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.5
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kelseyhightower/memkv v0.1.1
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/libp2p/go-reuseport v0.4.0
	github.com/mcuadros/go-version v0.0.0-20190830083331-035f6764e8d2
	github.com/mdlayher/netlink v1.7.2
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/natefinch/atomic v1.0.1
	github.com/nmrshll/go-cp v0.0.0-20180115193924-61436d3b7cfa
	github.com/nxadm/tail v1.4.11
	github.com/olekukonko/tablewriter v0.0.5
	github.com/olivere/elastic/v7 v7.0.31
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/ginkgo/v2 v2.22.2
	github.com/onsi/gomega v1.36.2
	github.com/oschwald/geoip2-golang v1.11.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/calico/lib/httpmachinery v0.0.0-00010101000000-000000000000
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/prometheus-community/elasticsearch_exporter v1.5.0
	github.com/prometheus/client_golang v1.21.1
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.63.0
	github.com/prometheus/procfs v0.15.1
	github.com/robfig/cron v1.2.0
	github.com/safchain/ethtool v0.5.10
	github.com/shirou/gopsutil/v4 v4.25.2
	github.com/sirupsen/logrus v1.9.3
	github.com/skeema/knownhosts v1.3.1
	github.com/slack-go/slack v0.16.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.0
	github.com/stretchr/testify v1.10.0
	github.com/tchap/go-patricia/v2 v2.3.2
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/tidwall/gjson v1.18.0
	github.com/tigera/api v0.0.0-20240708152828-da21375e20da
	github.com/tigera/windows-networking v0.0.0-20211112174220-6a90051af748
	github.com/urfave/cli/v2 v2.27.6
	github.com/vishvananda/netlink v1.3.1-0.20250206174618-62fb240731fa
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd/api/v3 v3.5.19
	go.etcd.io/etcd/client/pkg/v3 v3.5.19
	go.etcd.io/etcd/client/v2 v2.305.19
	go.etcd.io/etcd/client/v3 v3.5.19
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.36.0
	golang.org/x/net v0.37.0
	golang.org/x/oauth2 v0.28.0
	golang.org/x/sync v0.12.0
	golang.org/x/sys v0.31.0
	golang.org/x/text v0.23.0
	golang.org/x/time v0.11.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250313205543-e70fdf4c4cb4
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	// validator.v9 must be at v9.30.2 for libcalico-go to build. It may be possible to upgrade this
	// with some changes to libcalico-go, though.
	gopkg.in/go-playground/validator.v9 v9.30.2
	// Replaced with older version below until we can handle the updated permissions it now puts on log files.
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	// Most k8s.io modules we 'require' will also need a 'replace' directive below in order for the module graph to resolve.
	// Ensure that any version updates to k8s.io modules are reflected in the replace block if those modules require replacement.
	k8s.io/api v0.32.3
	k8s.io/apiextensions-apiserver v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/apiserver v0.32.3
	k8s.io/client-go v0.32.3
	k8s.io/component-base v0.32.3
	k8s.io/klog/v2 v2.130.1
	k8s.io/kube-aggregator v0.32.3
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f
	k8s.io/kubectl v0.32.3
	k8s.io/kubernetes v1.32.3
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738
	modernc.org/memory v1.8.2
	sigs.k8s.io/controller-runtime v0.20.3
	sigs.k8s.io/kind v0.27.0
	sigs.k8s.io/knftables v0.0.18
	sigs.k8s.io/network-policy-api v0.1.5
	sigs.k8s.io/yaml v1.4.0
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	cel.dev/expr v0.19.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/JeffAshton/win_pdh v0.0.0-20161109143554-76bb4ee9f0ab // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hnslib v0.0.8 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/alexflint/go-filemutex v1.3.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.62 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/acm v1.30.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.51.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.44.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.37.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.36.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecs v1.52.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.38.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.37.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/lambda v1.69.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/rds v1.91.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.46.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.69.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.34.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.57.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.33.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.37.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.56.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.29.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bi-zone/etw v0.0.0-20200916105032-b215904fae4f // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.3.9 // indirect
	github.com/cncf/xds/go v0.0.0-20241223141626-cff3c89139a3 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/containerd/containerd/api v1.8.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.6 // indirect
	github.com/containerd/typeurl/v2 v2.2.2 // indirect
	github.com/corazawaf/libinjection-go v0.2.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.8.2 // indirect
	github.com/elastic/go-sysinfo v1.13.1 // indirect
	github.com/elastic/go-ucfg v0.8.8 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/euank/go-kmsg-parser v2.0.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.6 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/form v3.1.4+incompatible // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.22.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.4 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/gonvenience/bunt v1.3.5 // indirect
	github.com/gonvenience/neat v1.3.12 // indirect
	github.com/gonvenience/term v1.0.2 // indirect
	github.com/gonvenience/text v1.0.7 // indirect
	github.com/gonvenience/wrap v1.1.2 // indirect
	github.com/gonvenience/ytbx v1.4.4 // indirect
	github.com/google/cadvisor v0.52.1 // indirect
	github.com/google/cel-go v0.22.0 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/gruntwork-io/go-commons v0.8.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/homeport/dyff v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/karrick/godirwalk v1.17.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/libopenstorage/openstorage v1.0.0 // indirect
	github.com/lithammer/dedent v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magefile/mage v1.15.1-0.20241126214340-bdc92f694516 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-ciede2000 v0.0.0-20170301095244-782e8c62fec3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-zglob v0.0.2-0.20190814121620-e3c945676326 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/cgroups v0.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opencontainers/runc v1.2.5 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/opencontainers/selinux v1.11.1 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/petar-dambovaliev/aho-corasick v0.0.0-20240411101913-e07a1f0e8eb4 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/pquerna/otp v1.4.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/urfave/cli v1.22.16 // indirect
	github.com/valllabh/ocsf-schema-golang v1.0.3 // indirect
	github.com/virtuald/go-ordered-json v0.0.0-20170621173500-b18e6e673d74 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.elastic.co/apm/module/apmzap/v2 v2.6.2 // indirect
	go.elastic.co/apm/v2 v2.6.2 // indirect
	go.elastic.co/fastjson v1.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful v0.46.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250207012021-f9890c6ad9f3 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/tools v0.29.0 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250106144421-5f5ef82da422 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gorm.io/driver/postgres v1.5.11 // indirect
	gorm.io/gorm v1.25.12 // indirect
	howett.net/plist v1.0.1 // indirect
	k8s.io/cloud-provider v0.32.3 // indirect
	k8s.io/component-helpers v0.32.3 // indirect
	k8s.io/controller-manager v0.32.3 // indirect
	k8s.io/cri-api v0.32.3 // indirect
	k8s.io/cri-client v0.32.3 // indirect
	k8s.io/csi-translation-lib v0.32.3 // indirect
	k8s.io/dynamic-resource-allocation v0.32.3 // indirect
	k8s.io/kms v0.32.3 // indirect
	k8s.io/kube-scheduler v0.32.3 // indirect
	k8s.io/kubelet v0.32.3 // indirect
	k8s.io/mount-utils v0.32.3 // indirect
	k8s.io/pod-security-admission v0.32.3 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.31.0 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
)

replace (
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/projectcalico/calico/lib/httpmachinery => ./lib/httpmachinery

	github.com/tigera/api => ./api

	// Newer versions set the file mode on logs to 0600, which breaks a lot of our tests.
	gopkg.in/natefinch/lumberjack.v2 => gopkg.in/natefinch/lumberjack.v2 v2.0.0

	// Need replacements for all the k8s subsidiary projects that are pulled in indirectly because
	// the kubernetes repo pulls them in via a replacement to its own vendored copies, which doesn't work for
	// transient imports.
	k8s.io/api => k8s.io/api v0.32.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.32.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.32.3
	k8s.io/apiserver => k8s.io/apiserver v0.32.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.32.3
	k8s.io/client-go => k8s.io/client-go v0.32.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.32.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.32.3
	k8s.io/code-generator => k8s.io/code-generator v0.32.3
	k8s.io/component-base => k8s.io/component-base v0.32.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.32.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.32.3
	k8s.io/cri-api => k8s.io/cri-api v0.32.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.32.3
	k8s.io/endpointslice => k8s.io/endpointslice v0.32.3
	k8s.io/externaljwt => k8s.io/externaljwt v0.32.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.32.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.32.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.32.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.32.3
	k8s.io/kubectl => k8s.io/kubectl v0.32.3
	k8s.io/kubelet => k8s.io/kubelet v0.32.3
	k8s.io/metrics => k8s.io/metrics v0.32.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.32.3
	k8s.io/node-api => k8s.io/node-api v0.32.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.32.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.32.3
)
