module github.com/projectcalico/kube-controllers

go 1.12

require (
	github.com/apparentlymart/go-cidr v1.0.1
	github.com/coreos/etcd v3.3.18+incompatible

	github.com/elastic/cloud-on-k8s v0.0.0-20190924084002-6ce4c9177aec
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mattn/go-isatty v0.0.8 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/tigera/api v0.0.0-20200115221514-2e8e59c327b0
	github.com/tigera/licensing v0.0.0-20200103202009-cd8a8b68f3ac
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vishvananda/netns v0.0.0-20170219233438-54f0e4339ce7 // indirect

	// k8s.io/api v1.16.3 is at 16d7abae0d2a
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb

	// k8s.io/apiserver 1.16.3 is at 9ca1dc586682
	k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible

	k8s.io/klog v1.0.0
)

replace (
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20200131043520-c21722bed5bb
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200130194212-1ef07d41ddd9
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200130203734-85760318d620
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200131043248-5d8befc81a65
	// We need to hold back prometheus/client_golang to avoid a build failure. This is hopefully a
	// temporary fix.
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.4
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)
