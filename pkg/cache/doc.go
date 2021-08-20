// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package cache

// This package caches all the selectors, labels, and the mapping between them.
// If either label matches a selector or label that previously matched a selector is no longer valid,
// it reports it back to the caller using call back function.
