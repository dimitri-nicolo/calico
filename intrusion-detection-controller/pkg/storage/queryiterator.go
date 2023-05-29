// Copyright 2019 Tigera Inc. All rights reserved.

package storage

import (
	"context"

	"github.com/projectcalico/calico/linseed/pkg/client"
)

type queryEntry[T any, V any] struct {
	key         QueryKey
	queryParams V
	listPager   client.ListPager[T]
	listFn      client.ListFunc[T]
}

type Iterator[T any] interface {
	Next() bool
	Value() (key QueryKey, val T)
	Err() error
}

type queryIterator[T any, V any] struct {
	queries []queryEntry[T, V]
	ctx     context.Context
	name    string
	hits    []T
	key     QueryKey
	val     T
	err     error
}

func (i *queryIterator[T, V]) Next() bool {
	for len(i.queries) > 0 {
		if len(i.hits) == 0 {
			entry := i.queries[0]
			i.key = entry.key
			results, errors := entry.listPager.Stream(i.ctx, entry.listFn)

			for page := range results {
				i.hits = append(i.hits, page.Items...)
			}

			err, ok := <-errors
			if ok {
				i.err = err
				return false
			}
			i.queries = i.queries[1:]
		}

		if len(i.hits) > 0 {
			i.val = i.hits[0]
			i.hits = i.hits[1:]
			return true
		}
	}

	for len(i.hits) > 0 {
		i.val = i.hits[0]
		i.hits = i.hits[1:]
		return true
	}

	return false
}

func (i *queryIterator[T, V]) Value() (QueryKey, T) {
	return i.key, i.val
}

func (i *queryIterator[T, V]) Err() error {
	return i.err
}

func newQueryIterator[T any, V any](ctx context.Context, queries []queryEntry[T, V], name string) Iterator[T] {
	return &queryIterator[T, V]{ctx: ctx, queries: queries, name: name}
}
