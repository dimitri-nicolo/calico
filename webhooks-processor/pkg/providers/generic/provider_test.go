// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package generic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericProviderValidation(t *testing.T) {
	p := GenericProvider{}
	t.Run("valid config", func(t *testing.T) {
		err := p.Validate(map[string]string{
			"url": "some-url",
		})
		require.NoError(t, err)
	})

	t.Run("no URL", func(t *testing.T) {
		err := p.Validate(map[string]string{})
		require.Error(t, err)
	})
}
