module github.com/projectcalico/calico

go 1.22.2

require (
	github.com/AppsFlyer/go-sundheit v0.6.0
	github.com/BurntSushi/toml v1.4.0
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.12.6
	github.com/PaloAltoNetworks/pango v0.10.2
	github.com/SermoDigital/jose v0.9.2-0.20161205224733-f6df55f235c2
	github.com/alecthomas/participle v0.7.1
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aquasecurity/kube-bench v0.6.19
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/aws/aws-sdk-go v1.55.5
	github.com/aws/aws-sdk-go-v2 v1.30.4
	github.com/aws/aws-sdk-go-v2/config v1.27.28
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.12
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.175.1
	github.com/aws/smithy-go v1.20.4
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f
	github.com/bronze1man/goStrongswanVici v0.0.0-20221114103242-3f6dc524986c
	github.com/buger/jsonparser v1.1.1
	github.com/caimeo/iniflags v0.0.0-20171110233946-ef4ae6c5cd79
	github.com/cnf/structhash v0.0.0-20201127153200-e1b16c1ebc08
	// pod2daemon/cisdriver build failure after upgrading to v1.10.0+
	github.com/container-storage-interface/spec v1.9.0
	github.com/containernetworking/cni v1.2.2
	github.com/containernetworking/plugins v1.5.1
	github.com/corazawaf/coraza-coreruleset v0.0.0-20231103220038-fd5c847140a6
	github.com/corazawaf/coraza/v3 v3.2.1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-semver v0.3.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dexidp/dex v0.0.0-20240603194342-23efe9200ccd
	github.com/dexidp/dex/api/v2 v2.1.1-0.20240603194342-23efe9200ccd
	github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/docker v27.1.1+incompatible
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/elastic/cloud-on-k8s/v2 v2.11.1
	github.com/elastic/go-elasticsearch/v7 v7.17.10
	github.com/envoyproxy/go-control-plane v0.13.0
	github.com/felixge/httpsnoop v1.0.4
	github.com/florianl/go-nfqueue v1.3.2
	github.com/fsnotify/fsnotify v1.7.0
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/ghodss/yaml v1.0.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-ini/ini v1.67.0
	github.com/go-jose/go-jose/v4 v4.0.4
	github.com/go-kit/log v0.2.1
	github.com/go-openapi/runtime v0.28.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gofrs/flock v0.12.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/googleapis v1.4.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/golang/protobuf v1.5.4
	github.com/golang/snappy v0.0.4
	github.com/google/btree v1.1.2
	github.com/google/go-cmp v0.6.0
	github.com/google/go-github/v53 v53.2.0
	github.com/google/gopacket v1.1.19
	github.com/google/netstack v0.0.0-20191123085552-55fcc16cd0eb
	github.com/google/safetext v0.0.0-20230106111101-7156a760e523
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/goyek/goyek/v2 v2.1.0
	github.com/goyek/x v0.1.7
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/gruntwork-io/terratest v0.47.0
	github.com/hashicorp/yamux v0.1.1
	github.com/ishidawataru/sctp v0.0.0-20230406120618-7ff4192f6ff2
	github.com/jarcoal/httpmock v1.3.1
	github.com/jcchavezs/mergefs v0.0.0-20230503083351-07f27d256761
	github.com/joho/godotenv v1.5.1
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.12
	github.com/juju/clock v1.1.1
	github.com/juju/errors v1.0.0
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.4.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kelseyhightower/memkv v0.1.1
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/libp2p/go-reuseport v0.4.0
	github.com/mcuadros/go-version v0.0.0-20190830083331-035f6764e8d2
	github.com/mdlayher/netlink v1.7.2
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/mitchellh/go-homedir v1.1.0
	github.com/natefinch/atomic v1.0.1
	github.com/nmrshll/go-cp v0.0.0-20180115193924-61436d3b7cfa
	github.com/nxadm/tail v1.4.11
	github.com/oklog/run v1.1.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/olivere/elastic/v7 v7.0.31
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/ginkgo/v2 v2.20.0
	github.com/onsi/gomega v1.34.1
	github.com/oschwald/geoip2-golang v1.11.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/prometheus-community/elasticsearch_exporter v1.5.0
	github.com/prometheus/client_golang v1.20.1
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.55.0
	github.com/prometheus/procfs v0.15.1
	github.com/rakelkar/gonetsh v0.3.2
	github.com/robfig/cron v1.2.0
	github.com/safchain/ethtool v0.4.1
	github.com/shirou/gopsutil v0.0.0-20190323131628-2cbc9195c892
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	github.com/tchap/go-patricia/v2 v2.3.1
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/tidwall/gjson v1.17.3
	github.com/tigera/api v0.0.0-20240708152828-da21375e20da
	github.com/tigera/windows-networking v0.0.0-20211112174220-6a90051af748
	github.com/vishvananda/netlink v1.2.1-beta.2.0.20240703200800-b54f85093f4a
	github.com/willf/bitset v1.1.11
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd/api/v3 v3.5.15
	go.etcd.io/etcd/client/pkg/v3 v3.5.15
	go.etcd.io/etcd/client/v2 v2.305.15
	go.etcd.io/etcd/client/v3 v3.5.15
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.26.0
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	golang.org/x/net v0.28.0
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.24.0
	golang.org/x/text v0.17.0
	golang.org/x/time v0.6.0
	// package golang.zx2c4.com/wireguard/ipc/namedpipe is required to be v0.0.20200121
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/elazarl/goproxy.v1 v1.0.0-20180725130230-947c36da3153
	// validator.v9 must be at v9.30.2 for libcalico-go to build. It may be possible to upgrade this
	// with some changes to libcalico-go, though.
	gopkg.in/go-playground/validator.v9 v9.30.2
	// Replaced with older version below until we can handle the updated permissions it now puts on log files.
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/tchap/go-patricia.v2 v2.3.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.29.8
	k8s.io/apiextensions-apiserver v0.29.8
	k8s.io/apimachinery v0.29.8
	k8s.io/apiserver v0.29.8
	k8s.io/client-go v0.29.8
	k8s.io/code-generator v0.29.8
	k8s.io/component-base v0.29.8
	k8s.io/klog/v2 v2.120.1
	k8s.io/kube-aggregator v0.29.8
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00
	k8s.io/kubectl v0.28.9
	k8s.io/kubernetes v1.29.8
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57
	modernc.org/memory v1.8.0
	sigs.k8s.io/controller-runtime v0.16.6
	sigs.k8s.io/kind v0.24.0
	sigs.k8s.io/knftables v0.0.17
	sigs.k8s.io/yaml v1.4.0
)

require (
	cloud.google.com/go/auth v0.4.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/mocks v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/GoogleCloudPlatform/k8s-cloud-provider v1.18.1-0.20220218231025-f11817397a1b // indirect
	github.com/JeffAshton/win_pdh v0.0.0-20161109143554-76bb4ee9f0ab // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230217124315-7d5c6f04bbb8 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/alexflint/go-filemutex v1.3.0 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/armon/circbuf v0.0.0-20150827004946-bbbad097214e // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.29.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.4 // indirect
	github.com/beevik/etree v1.4.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bi-zone/etw v0.0.0-20200916105032-b215904fae4f // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/checkpoint-restore/go-criu/v5 v5.3.0 // indirect
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/cloudflare/circl v1.3.9 // indirect
	github.com/cncf/xds/go v0.0.0-20240423153145-555b57ec207b // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/containerd/console v1.0.4 // indirect
	github.com/containerd/errdefs v0.1.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.5 // indirect
	github.com/corazawaf/libinjection-go v0.2.1 // indirect
	github.com/coreos/go-iptables v0.7.0 // indirect
	github.com/coreos/go-oidc/v3 v3.10.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-ucfg v0.8.6 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20240719135048-6ca80f564554 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20240719135048-6ca80f564554 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/euank/go-kmsg-parser v2.0.0+incompatible // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.5 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-ldap/ldap/v3 v3.4.6 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/glog v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/gonvenience/bunt v1.3.5 // indirect
	github.com/gonvenience/neat v1.3.12 // indirect
	github.com/gonvenience/term v1.0.2 // indirect
	github.com/gonvenience/text v1.0.7 // indirect
	github.com/gonvenience/wrap v1.1.2 // indirect
	github.com/gonvenience/ytbx v1.4.4 // indirect
	github.com/google/cadvisor v0.48.1 // indirect
	github.com/google/cel-go v0.17.7 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.4 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/gruntwork-io/go-commons v0.8.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/homeport/dyff v1.6.0 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.4 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/karrick/godirwalk v1.17.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/libopenstorage/openstorage v1.0.0 // indirect
	github.com/lithammer/dedent v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-ciede2000 v0.0.0-20170301095244-782e8c62fec3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/mattn/go-zglob v0.0.2-0.20190814121620-e3c945676326 // indirect
	github.com/mdlayher/genetlink v1.0.0 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb // indirect
	github.com/mrunalp/fileutils v0.5.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runc v1.1.12 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/petar-dambovaliev/aho-corasick v0.0.0-20240411101913-e07a1f0e8eb4 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/pquerna/otp v1.2.0 // indirect
	github.com/rivo/uniseg v0.1.0 // indirect
	github.com/rubiojr/go-vhd v0.0.0-20200706105327-02e210299021 // indirect
	github.com/russellhaering/goxmldsig v1.4.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/seccomp/libseccomp-golang v0.10.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/urfave/cli v1.22.15 // indirect
	github.com/virtuald/go-ordered-json v0.0.0-20170621173500-b18e6e673d74 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/vmware/govmomi v0.30.6 // indirect
	go.elastic.co/apm/module/apmzap/v2 v2.4.7 // indirect
	go.elastic.co/apm/v2 v2.4.7 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful v0.46.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/term v0.23.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	golang.zx2c4.com/wireguard v0.0.20200121 // indirect
	google.golang.org/api v0.182.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gorm.io/driver/postgres v1.4.6 // indirect
	gorm.io/gorm v1.25.1 // indirect
	howett.net/plist v1.0.0 // indirect
	k8s.io/cloud-provider v0.29.8 // indirect
	k8s.io/component-helpers v0.29.8 // indirect
	k8s.io/controller-manager v0.29.8 // indirect
	k8s.io/cri-api v0.29.8 // indirect
	k8s.io/csi-translation-lib v0.28.9 // indirect
	k8s.io/dynamic-resource-allocation v0.28.9 // indirect
	k8s.io/gengo v0.0.0-20230829151522-9cce18d56c01 // indirect
	k8s.io/kms v0.29.8 // indirect
	k8s.io/kube-scheduler v0.29.8 // indirect
	k8s.io/kubelet v0.28.9 // indirect
	k8s.io/legacy-cloud-providers v0.29.8 // indirect
	k8s.io/mount-utils v0.28.9 // indirect
	k8s.io/pod-security-admission v0.29.8 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.28.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

replace (
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.18.0
	github.com/prometheus/common => github.com/prometheus/common v0.47.0

	github.com/tigera/api => ./api

	// Newer versions set the file mode on logs to 0600, which breaks a lot of our tests.
	gopkg.in/natefinch/lumberjack.v2 => gopkg.in/natefinch/lumberjack.v2 v2.0.0

	// Need replacements for all the k8s subsidiary projects that are pulled in indirectly because
	// the kubernetes repo pulls them in via a replacement to its own vendored copies, which doesn't work for
	// transient imports.
	k8s.io/api => k8s.io/api v0.29.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.29.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.29.8
	k8s.io/apiserver => k8s.io/apiserver v0.29.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.29.8
	k8s.io/client-go => k8s.io/client-go v0.29.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.29.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.29.8
	k8s.io/code-generator => k8s.io/code-generator v0.29.8
	k8s.io/component-base => k8s.io/component-base v0.29.8
	k8s.io/component-helpers => k8s.io/component-helpers v0.29.8
	k8s.io/controller-manager => k8s.io/controller-manager v0.29.8
	k8s.io/cri-api => k8s.io/cri-api v0.29.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.29.8
	k8s.io/endpointslice => k8s.io/endpointslice v0.29.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.29.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.29.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.29.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.29.8
	k8s.io/kubectl => k8s.io/kubectl v0.29.8
	k8s.io/kubelet => k8s.io/kubelet v0.29.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.29.8
	k8s.io/metrics => k8s.io/metrics v0.29.8
	k8s.io/mount-utils => k8s.io/mount-utils v0.29.8
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.29.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.29.8
)
