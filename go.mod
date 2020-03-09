module github.com/tigera/license-agent

go 1.12

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200203232830-a5de53a78f58

require (
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v2.5.1+incompatible
	github.com/tigera/ts-queryserver v2.6.2+incompatible
)
