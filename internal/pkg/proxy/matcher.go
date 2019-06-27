// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	"github.com/tigera/voltron/internal/pkg/targets"
)

// Matcher selects a target URL based on a predefined pattern
type Matcher interface {
	Match(request *http.Request) (*url.URL, error)
}

// PathMatcher matches against the path
type PathMatcher struct {
	targets *targets.Targets
}

// Match returns a target URL using the given URL path value against different patterns. The value returned is the full URL
// or an error if no match is found
func (matcher *PathMatcher) Match(request *http.Request) (*url.URL, error) {
	path := request.URL.Path
	return Match(path, matcher.targets.List())
}

// NewPathMatcher creates a new PathMatcher
func NewPathMatcher(tgt *targets.Targets) *PathMatcher {
	return &PathMatcher{
		targets: tgt,
	}
}

// Match returns a target URL matched against different patterns. The value returned is the full URL
// It will return an error if no match is found or an empty value is being passed
func Match(value string, targets map[string]*url.URL) (*url.URL, error) {
	if len(value) == 0 {
		return nil, errors.Errorf("could not extract target from request")
	}

	for pattern := range targets {
		match, err := regexp.MatchString(pattern, value)
		if err != nil {
			continue
		} else if match {
			return targets[pattern], nil
		}
	}

	return nil, errors.Errorf("configuration missing for path %#v", value)
}
