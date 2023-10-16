// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleValidConfig() map[string]string {
	return map[string]string{
		"url":       "some-url",
		"project":   "test",
		"issueType": "test",
		"username":  "test",
		"apiToken":  "test",
	}
}
func TestJiraProviderValidation(t *testing.T) {
	p := Jira{}
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

	t.Run("no url", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "url")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no project", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "project")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no issueType", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "issueType")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no username", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "username")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no apiToken", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "apiToken")

		err := p.Validate(c)
		require.Error(t, err)
	})
}
