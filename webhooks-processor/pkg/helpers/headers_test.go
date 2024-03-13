// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
)

func TestProcessHeaders(t *testing.T) {
	// empty corner case:
	rawHeaders := ""
	headers, err := helpers.ProcessHeaders(rawHeaders)
	assert.NoError(t, err)
	assert.Equal(t, headers, map[string]string{})

	// valid arbitrary data:
	rawHeaders = `
this is not a header line and will be ignored as well as the next empty line

this-is-a-header:with some data in it
this-is-another-header:with-data-and-colon:like-that`
	headers, err = helpers.ProcessHeaders(rawHeaders)
	assert.NoError(t, err)
	assert.Equal(t, headers, map[string]string{
		"this-is-a-header":       "with some data in it",
		"this-is-another-header": "with-data-and-colon:like-that",
	})

	// invalid header name
	rawHeaders = "this-is-a-`header`:with invalid characters in its name"
	_, err = helpers.ProcessHeaders(rawHeaders)
	assert.Error(t, err)

	// invalid header value
	rawHeaders = "this-is-a-header:with invalid characters in its value ™"
	_, err = helpers.ProcessHeaders(rawHeaders)
	assert.Error(t, err)
}
