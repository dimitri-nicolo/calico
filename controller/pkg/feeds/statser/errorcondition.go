// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	lru "github.com/hashicorp/golang-lru"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type ErrorConditions struct {
	c *lru.Cache
}

func NewErrorConditions(size int) (*ErrorConditions, error) {
	l, err := lru.New(size)
	return &ErrorConditions{
		c: l,
	}, err
}

func (e *ErrorConditions) Add(t string, err error) {
	e.c.Add(v3.ErrorCondition{Type: t, Message: err.Error()}, nil)
}

func (e *ErrorConditions) Clear(t string) {
	for _, i := range e.c.Keys() {
		if i.(v3.ErrorCondition).Type == t {
			e.c.Remove(i)
		}
	}
}

func (e *ErrorConditions) Errors() (res []v3.ErrorCondition) {
	for _, i := range e.c.Keys() {
		res = append(res, i.(v3.ErrorCondition))
	}

	return
}

func (e *ErrorConditions) TypedErrors(t string) (res []v3.ErrorCondition) {
	for _, i := range e.c.Keys() {
		if i.(v3.ErrorCondition).Type == t {
			res = append(res, i.(v3.ErrorCondition))
		}
	}

	return
}
