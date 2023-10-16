// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleValidConfig() map[string]string {
	return map[string]string{
		"url": "https://hooks.slack.com/test-hook",
	}
}

func TestSlackProviderValidation(t *testing.T) {
	p := Slack{}
	t.Run("valid config", func(t *testing.T) {
		err := p.Validate(sampleValidConfig())
		require.NoError(t, err)
	})

	t.Run("no url", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "url")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("not a slack url", func(t *testing.T) {
		c := sampleValidConfig()
		c["url"] = "https://somethinng-thats-not-slack.com/test"

		err := p.Validate(c)
		require.Error(t, err)
	})
}
