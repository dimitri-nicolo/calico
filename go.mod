module github.com/projectcalico/kube-controllers

go 1.12

require (
	github.com/Azure/go-autorest/autorest v0.9.3 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1 // indirect
	github.com/Masterminds/semver v1.2.2 // indirect
	github.com/apparentlymart/go-cidr v1.0.1
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/tigera/licensing v2.5.1+incompatible
	github.com/vishvananda/netns v0.0.0-20170219233438-54f0e4339ce7 // indirect

	// k8s.io/api v1.16.3 is at 16d7abae0d2a
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb

	// k8s.io/apiserver 1.16.3 is at 9ca1dc586682
	k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v0.0.0-20191114101535-6c5935290e33

	k8s.io/klog v1.0.0
)

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200111223629-729cb7baf6f1
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200110190915-9fa812d46e44
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200103211238-4018e3107793
	// We need to hold back prometheus/client_golang to avoid a build failure. This is hopefully a
	// temporary fix.
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.4
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
