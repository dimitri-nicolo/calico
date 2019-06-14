// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package targets

import (
	"net/url"
)

func parse(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	return url
}
