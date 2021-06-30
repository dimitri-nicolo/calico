module github.com/tigera/es-gateway

go 1.15

require (
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/gorilla/mux v1.7.3
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.10.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
