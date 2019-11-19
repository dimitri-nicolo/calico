package elastic_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/projectcalico/libcalico-go/lib/testutils"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/tigera/lma/pkg/elastic"
)

func TestElastic(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elastic Suite")
}

func testIndexSettings(cfg *Config, index string, settings map[string]string) {
	c, err := getESClient(cfg)
	Expect(err).ToNot(HaveOccurred())

	for key, value := range settings {
		Eventually(func() (interface{}, error) {
			return getIndexSetting(c, index, key)
		}, 10, 1).Should(Equal(value))
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

func getIndexSetting(client *elastic.Client, index string, setting string) (interface{}, error) {
	settings, err := client.IndexGetSettings(index).Do(context.Background())
	if err != nil {
		return "", err
	}
	indexSettings := settings[index].Settings["index"].(map[string]interface{})
	return indexSettings[setting], nil
}
