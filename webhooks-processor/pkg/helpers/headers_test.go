// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package helpers_test

import (
	"testing"

	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/stretchr/testify/assert"
)

func TestProcessHeaders(t *testing.T) {
	// empty corner case:
	rawHeaders := ""
	headers := helpers.ProcessHeaders(rawHeaders)
	assert.Equal(t, headers, map[string]string{})

	// arbitrary data:
	rawHeaders = `
this is not a header line and will be ignored as well as the next empty line

this-is-a-header:with some data in it
this-is-another-header:with-data-and-colon:like-that`
	headers = helpers.ProcessHeaders(rawHeaders)
	assert.Equal(t, headers, map[string]string{
		"this-is-a-header":       "with some data in it",
		"this-is-another-header": "with-data-and-colon:like-that",
	})
}
