// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build !tesla
// +build !tesla

package elastic

// AddIndexInfix is a hook to add any extra substring to the index pattern. For Enterprise, we
// currently do not add any extra infix.
func AddIndexInfix(index string) string {
	return index
}
