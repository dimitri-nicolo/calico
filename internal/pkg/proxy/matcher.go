// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy

import (
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	"github.com/tigera/voltron/internal/pkg/targets"
)

type Matcher interface {
	Match(request *http.Request) (*url.URL, error)
}

type PathMatcher struct {
	targets *targets.Targets
}

// Matches path value against different patterns. The value returned is the full URL or an error
// if no match is found
func (matcher *PathMatcher) Match(request *http.Request) (*url.URL, error) {
	path := request.URL.Path
	return Match(path, matcher.targets.List())
}

func NewPathMatcher(tgt *targets.Targets) *PathMatcher {
	return &PathMatcher{
		targets: tgt,
	}
}

type HeaderMatcher struct {
	targets *targets.Targets
	header  string
}

// Matches header value against different patterns. The value returned is the full URL or an error
// if no match is found
func (matcher HeaderMatcher) Match(request *http.Request) (*url.URL, error) {
	if hasMultipleValues(matcher.header, request) {
		return nil, errors.New("could not determine target to select")
	}
	header := request.Header.Get(matcher.header)
	return Match(header, matcher.targets.List())
}

func NewHeaderMatcher(tgt *targets.Targets, header string) *HeaderMatcher {
	return &HeaderMatcher{
		targets: tgt,
		header:  header,
	}
}

func hasMultipleValues(header string, request *http.Request) bool {
	return len(request.Header[textproto.CanonicalMIMEHeaderKey(header)]) != 1
}

// Matches a value against different patterns. The value returned is the full URL
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
