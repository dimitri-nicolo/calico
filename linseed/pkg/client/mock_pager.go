package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockListPager is useful for tests in order to return predetermined responses.
// Pass in a custom ListFunc to return objects of type T each time it is called.
func NewMockListPager[T any](p v1.Params, f ListFunc[T], opts ...ListPagerOption[T]) ListPager[T] {
	return &mockListPager[T]{
		params:    p,
		listFn:    f,
		realPager: NewListPager(p, opts...),
	}
}

type mockListPager[T any] struct {
	// We include a "real" list pager to do the heavy lifting, however
	// we will intercept and replace the listFunc provided and use the mock
	// listFunc that we were initialized with.
	realPager ListPager[T]
	params    v1.Params
	listFn    ListFunc[T]
}

func (m *mockListPager[T]) PageFunc(_ ListFunc[T]) PageFunc[T] {
	return m.realPager.PageFunc(m.listFn)
}

func (m *mockListPager[T]) Stream(ctx context.Context, _ ListFunc[T]) (<-chan v1.List[T], <-chan error) {
	return m.realPager.Stream(ctx, m.listFn)
}
