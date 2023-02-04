// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// NewListPager returns a helper for performing paged list operations against the Linseed API.
//
// Example usage:
//
// listFunc := client.L3Flows().List
// pager := NewListPager[v1.L3Flow](&v1.L3FlowParams{})
// getPage := pager.PageFunc(listFunc)
// results, more, err := getPage()
func NewListPager[T any](p v1.Params) ListPager[T] {
	return &listPager[T]{
		params: p,
	}
}

// ListPager manages performing paged lists against the Linseed API.
type ListPager[T any] interface {
	PageFunc(ListFunc[T]) PageFunc[T]
	Stream(context.Context, ListFunc[T]) (<-chan v1.List[T], <-chan error)
}

// ListFunc is a function that returns a page of items.
type ListFunc[T any] func(context.Context, v1.Params) (*v1.List[T], error)

// PageFunc can be called to return a single page of results and a boolean indicating if there
// are more pages available. Subsequent calls to PageFunc will return the next page of results.
// PageFunc will return true if there are more pages available.
type PageFunc[T any] func() (*v1.List[T], bool, error)

type listPager[T any] struct {
	params v1.Params
}

func (p *listPager[T]) PageFunc(f ListFunc[T]) PageFunc[T] {
	var afterKey map[string]interface{}
	var done bool

	return func() (*v1.List[T], bool, error) {
		if done {
			return nil, false, fmt.Errorf("PageFunc called after it was done")
		}
		p.params.SetAfterKey(afterKey)
		list, err := f(context.TODO(), p.params)
		if err != nil {
			done = true
			return nil, false, err
		}
		afterKey = list.AfterKey
		done = afterKey == nil
		return list, !done, nil
	}
}

// Stream starts a background routine which performs pagination and streams the results to the
// client using a channel. The stream will be terminated at the first sign of error.
// If you want full control over when new pages are sent, use PageFunc instead.
func (p *listPager[T]) Stream(ctx context.Context, f ListFunc[T]) (<-chan v1.List[T], <-chan error) {
	// For results, we allow some small amount of buffering, but won't buffer too much.
	// If the client is having trouble reading quickly enough, there's no point in retrieving
	// more pages.
	results := make(chan v1.List[T], 2)

	// We exit the gorouting on the first sign of error, so no need for buffering.
	errors := make(chan error)

	// Start a go routine which performs the paging and populates the
	// above channels with results.
	go func() {
		pageFunc := p.PageFunc(f)
		var err error
		var more bool
		var page *v1.List[T]
		defer func() {
			logrus.Debugf("stream goroutine completed, closing channels")
			close(results)
			close(errors)
		}()
		i := 0
		for {
			select {
			case <-ctx.Done():
				// Context canceled.
				return
			default:
				// Read pages and send them over the channel.
				logrus.Debugf("Calling PageFunc for page #%d", i)
				page, more, err = pageFunc()
				i++
				if err != nil {
					errors <- err
					return
				} else if page != nil {
					results <- *page
				}

				if !more {
					return
				}
			}
		}
	}()

	return results, errors
}
