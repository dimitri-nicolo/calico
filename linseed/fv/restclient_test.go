// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func TestFV_RESTClient(t *testing.T) {
	cluster := "cluster"
	tenant := ""

	// Build a basic RESTClient.
	cfg := rest.Config{
		CACertPath: "cert/RootCA.crt",
		URL:        "https://localhost:8444/",
	}
	rc, err := rest.NewClient(tenant, cfg)
	require.NoError(t, err)

	t.Run("should handle an OK response", func(t *testing.T) {
		// Build and send a request.
		params := v1.L3FlowParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}
		flows := v1.List[v1.L3Flow]{}

		err = rc.Post().
			Path("/flows").
			Cluster(cluster).
			Params(&params).
			Do(context.TODO()).
			Into(&flows)
		require.NoError(t, err)
	})

	t.Run("should handle a response with an error", func(t *testing.T) {
		// Build and send a request with no params - this should result
		// in an error, since it is missing required fields.
		params := v1.L3FlowParams{}
		flows := v1.List[v1.L3Flow]{}
		err = rc.Post().
			Path("/flows").
			Cluster(cluster).
			Params(&params).
			Do(context.TODO()).
			Into(&flows)
		require.Error(t, err)
	})

	t.Run("should handle a 404 response", func(t *testing.T) {
		// Build and send a request.
		params := v1.L3FlowParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}
		flows := v1.List[v1.L3Flow]{}

		err = rc.Post().
			Path("/bad/url").
			Cluster(cluster).
			Params(&params).
			Do(context.TODO()).
			Into(&flows)
		require.Error(t, err)
	})
}
