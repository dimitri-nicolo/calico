// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic_test

import (
	"context"
	"net/http"
	"time"

	es "github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/deep-packet-inspection/pkg/config"
	"github.com/tigera/deep-packet-inspection/pkg/elastic"
)

var _ = Describe("Elastic Search Forwarder", func() {
	var mockESClient *elastic.MockClient
	var cfg *config.Config
	var ctx context.Context
	mockClientFn := func(esCLI *es.Client, elasticIndexSuffix string) elastic.Client {
		return mockESClient
	}

	BeforeEach(func() {
		mockESClient = &elastic.MockClient{}
		mockESClient.AssertExpectations(GinkgoT())

		cfg = &config.Config{}
		ctx = context.Background()
	})

	It("should not retry after successfully indexing the document to ElasticSearch", func() {
		esForwarder, err := elastic.NewESForwarder(cfg, mockClientFn, 1*time.Second)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		mockESClient.On("Upsert", ctx, mock.Anything, mock.Anything).Return(nil).Times(1)

		esForwarder.Forward(elastic.EventData{ID: "", Doc: elastic.Doc{}})
	})

	It("should retry sending the document on connection error", func() {
		esForwarder, err := elastic.NewESForwarder(cfg, mockClientFn, 1*time.Second)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		numberOfCallsToSend := 0
		mockESClient.On("Upsert", ctx, mock.Anything, mock.Anything).Run(
			func(args mock.Arguments) {
				numberOfCallsToSend++
				for _, c := range mockESClient.ExpectedCalls {
					if c.Method == "Upsert" {
						if numberOfCallsToSend < 3 {
							c.ReturnArguments = mock.Arguments{&es.Error{Status: http.StatusForbidden}}
						} else {
							c.ReturnArguments = mock.Arguments{nil}
						}
					}
				}
			}).Times(3)

		esForwarder.Forward(elastic.EventData{ID: "", Doc: elastic.Doc{}})
	})

})
