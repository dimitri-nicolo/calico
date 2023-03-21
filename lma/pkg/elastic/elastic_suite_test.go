package elastic_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/projectcalico/calico/lma/pkg/elastic"
)

func TestElastic(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elastic Suite")
}

func deleteIndex(cfg *Config, index string) {
	client, err := getESClient(cfg)
	Expect(err).ToNot(HaveOccurred())
	_, err = client.DeleteIndex(index + "*").Do(context.Background())
	if err != nil {
		Expect(err).ToNot(HaveOccurred())
	}
}

func getESClient(cfg *Config) (*elastic.Client, error) {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(cfg.ParsedElasticURL.String()),
		elastic.SetHttpClient(&http.Client{}),
		elastic.SetSniff(false),
	}

	return elastic.NewClient(options...)
}
