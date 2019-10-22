module github.com/tigera/fluentd-docker/eks

go 1.13

require (
	github.com/aws/aws-sdk-go v1.25.8
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic v6.2.23+incompatible
	github.com/olivere/elastic/v7 v7.0.7
	github.com/onsi/gomega v1.7.0
	github.com/sirupsen/logrus v1.4.2
	k8s.io/apiserver v0.0.0-20191018030144-550b75f0da71
)
