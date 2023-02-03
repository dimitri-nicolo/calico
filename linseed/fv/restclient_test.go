package fv

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/stretchr/testify/require"
)

func TestFV_RESTClient(t *testing.T) {
	cluster := "cluster"
	tenant := ""

	// Build a basic RESTClient.
	cfg := rest.Config{
		CACertPath: "cert/RootCA.crt",
		URL:        fmt.Sprintf("https://localhost:8444/"),
	}
	rc, err := rest.NewClient(cluster, tenant, cfg)
	require.NoError(t, err)

	t.Run("should handle an OK response", func(t *testing.T) {
		// Build and send a request.
		params := v1.L3FlowParams{
			QueryParams: &v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}
		flows := v1.List[v1.L3Flow]{}

		err = rc.Post().
			Path("/api/v1/flows/network").
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
			Path("/api/v1/flows/network").
			Params(&params).
			Do(context.TODO()).
			Into(&flows)
		require.Error(t, err)
	})

	t.Run("should handle a 404 response", func(t *testing.T) {
		// Build and send a request.
		params := v1.L3FlowParams{
			QueryParams: &v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}
		flows := v1.List[v1.L3Flow]{}

		err = rc.Post().
			Path("/bad/url").
			Params(&params).
			Do(context.TODO()).
			Into(&flows)
		require.Error(t, err)
	})
}
