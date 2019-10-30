module github.com/tigera/lma

go 1.11

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191029214447-cf245bbeba6d // indirect

require (
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190324105220-f881eae9ec04
	k8s.io/client-go v8.0.0+incompatible
)
