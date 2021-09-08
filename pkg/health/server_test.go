// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

type testPinger struct{}

func (p testPinger) Ping(context.Context) error {
	return nil
}

type testReadier struct {
	r bool
}

func (r testReadier) Ready() bool {
	return r.r
}

func TestLiveness_ServeHTTP(t *testing.T) {
	RegisterTestingT(t)

	uut := liveness{testPinger{}}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/liveness", nil)
	uut.ServeHTTP(resp, req)
	Expect(resp.Code).To(Equal(http.StatusOK))
}

func TestReadiness_ServeHTTP(t *testing.T) {
	RegisterTestingT(t)

	uut := readiness{testReadier{true}}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/liveness", nil)
	uut.ServeHTTP(resp, req)
	Expect(resp.Code).To(Equal(http.StatusOK))

	uut.readier = testReadier{false}
	resp = httptest.NewRecorder()
	uut.ServeHTTP(resp, req)
	Expect(resp.Code).To(Equal(http.StatusInternalServerError))
}
