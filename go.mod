module github.com/tigera/es-gateway

go 1.15

require (
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
