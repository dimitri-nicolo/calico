module github.com/tigera/licensing

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-sql-driver/mysql v1.4.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211202172725-179fe7fe73ab
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909
)
