module github.com/tigera/prometheus-service

go 1.15

require (
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/jinzhu/copier v0.3.2 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/common v0.29.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	golang.org/x/sys v0.0.0-20210611083646-a4fc73990273 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiserver v0.21.1 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210601162810-75401bf3ee61
