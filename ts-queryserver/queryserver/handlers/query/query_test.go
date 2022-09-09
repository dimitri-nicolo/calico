// Copyright (c) 2022 Tigera. All rights reserved.
package query_test

import (
	_ "embed"
	"fmt"
	"io"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/query"
)

var (
	//go:embed testdata/expected_metrics.txt
	expectedMetrics string
)

var _ = Describe("Queryserver query metrics test", func() {

	It("should export metrics", func() {
		qi := client.MockQueryInterface{}
		qi.On("RunQuery", mock.Anything, mock.Anything).Return(&client.QueryClusterResp{
			NumHostEndpoints:                1,
			NumUnlabelledHostEndpoints:      2,
			NumUnprotectedHostEndpoints:     3,
			NumWorkloadEndpoints:            4,
			NumUnlabelledWorkloadEndpoints:  5,
			NumUnprotectedWorkloadEndpoints: 6,
			NumFailedWorkloadEndpoints:      7,
			NamespaceCounts: map[string]client.QueryClusterNamespaceCounts{
				"ns1": {
					NumWorkloadEndpoints:            8,
					NumUnlabelledWorkloadEndpoints:  9,
					NumUnprotectedWorkloadEndpoints: 10,
					NumFailedWorkloadEndpoints:      11,
				},
				"ns2": {
					NumWorkloadEndpoints:            12,
					NumUnlabelledWorkloadEndpoints:  13,
					NumUnprotectedWorkloadEndpoints: 14,
					NumFailedWorkloadEndpoints:      15,
				},
			},
		}, nil)

		r := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		q := query.NewQuery(&qi)
		q.Metrics(w, r)

		resp := w.Result()
		Expect(resp).NotTo(BeNil())
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(body)).To(Equal(expectedMetrics))
	})

	It("should write nothing when query interface failed to query", func() {
		qi := client.MockQueryInterface{}
		qi.On("RunQuery", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("RunQuery failed"))

		r := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		q := query.NewQuery(&qi)
		q.Metrics(w, r)

		resp := w.Result()
		Expect(resp).NotTo(BeNil())
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(body)).To(Equal(""))
	})

	It("should write nothing when response isn't of type QueryClusterResp", func() {
		qi := client.MockQueryInterface{}
		qi.On("RunQuery", mock.Anything, mock.Anything).Return(nil, nil)

		r := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		q := query.NewQuery(&qi)
		q.Metrics(w, r)

		resp := w.Result()
		Expect(resp).NotTo(BeNil())
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(body)).To(Equal(""))
	})

})
