module github.com/tigera/licensing

go 1.15

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gopkg.in/square/go-jose.v2 v2.2.3-0.20190111193340-cbf0fd6a984a
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210428213214-15c1878d8dc3
