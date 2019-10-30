module github.com/tigera/voltron

go 1.11

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191029214447-cf245bbeba6d

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/projectcalico/libcalico-go v0.0.0
	github.com/prometheus/client_golang v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/calico-k8sapiserver v2.6.0-0.dev.0.20190725235114-6b558a850bd6+incompatible
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190402105105-9b20910895af // indirect
	k8s.io/client-go v8.0.0+incompatible
)
