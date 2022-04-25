// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic_test

import (
	"context"
	"net/http"
	"time"

	lma "github.com/tigera/lma/pkg/elastic"

	es "github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/deep-packet-inspection/pkg/elastic"
)

var _ = Describe("Elastic Search Forwarder", func() {
	var mockESClient *lma.MockClient
	var ctx context.Context

	BeforeEach(func() {
		mockESClient = &lma.MockClient{}
		mockESClient.AssertExpectations(GinkgoT())

		ctx = context.Background()
	})

	It("should not retry after successfully indexing the document to ElasticSearch", func() {
		esForwarder, err := elastic.NewESForwarder(mockESClient, 1*time.Second)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		mockESClient.On("PutSecurityEventWithID", ctx, mock.Anything, mock.Anything).Return(nil, nil).Times(1)
		esForwarder.Forward(elastic.SecurityEvent{})
	})

	It("should retry sending the document on connection error", func() {
		esForwarder, err := elastic.NewESForwarder(mockESClient, 1*time.Second)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		numberOfCallsToSend := 0
		mockESClient.On("PutSecurityEventWithID", ctx, mock.Anything, mock.Anything).Run(
			func(args mock.Arguments) {
				numberOfCallsToSend++
				for _, c := range mockESClient.ExpectedCalls {
					if c.Method == "PutSecurityEventWithID" {
						if numberOfCallsToSend < 3 {
							c.ReturnArguments = mock.Arguments{nil, &es.Error{Status: http.StatusForbidden}}
						} else {
							c.ReturnArguments = mock.Arguments{nil, nil}
						}
					}
				}
			}).Times(3)
		esForwarder.Forward(elastic.SecurityEvent{})
	})

})
