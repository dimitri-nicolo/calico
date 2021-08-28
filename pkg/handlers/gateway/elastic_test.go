package gateway_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/tigera/es-gateway/pkg/handlers/gateway"
	"github.com/tigera/es-gateway/pkg/metrics"
	"github.com/tigera/es-gateway/pkg/middlewares"
)

type collectorMocker struct {
	mock.Mock
	metrics.Collector
}

func (c collectorMocker) CollectLogBytesWritten(tenantID, clusterID string, bytes float64) error {
	c.Called(tenantID, clusterID, bytes)
	return nil
}

func (c collectorMocker) CollectLogBytesRead(tenantID, clusterID string, bytes float64) error {
	c.Called(tenantID, clusterID, bytes)
	return nil
}

func (c collectorMocker) Serve(address string) error {
	c.Called(address)
	return nil
}

var _ = Describe("Test the elastic response hook", func() {

	const (
		clusterID = "my-cluster"
		tenantID  = "my-tenant"
	)
	var collector collectorMocker

	BeforeEach(func() {
		collector = collectorMocker{}
	})

	It("should call the metrics collector", func() {

		collector.On("CollectLogBytesRead", tenantID, clusterID, mock.Anything).Return(nil)
		collector.On("CollectLogBytesWritten", tenantID, clusterID, mock.Anything).Return(nil)

		fn := gateway.ElasticModifyResponseFunc(collector)
		req := &http.Request{RequestURI: "/some-uri", ContentLength: 25}
		req = req.WithContext(context.WithValue(context.WithValue(context.TODO(), middlewares.ClusterIDKey, clusterID), middlewares.TenantIDKey, tenantID))
		resp := &http.Response{
			Request:       req,
			ContentLength: 50,
			StatusCode:    200,
		}
		Expect(fn(resp)).NotTo(HaveOccurred())
	})
})
