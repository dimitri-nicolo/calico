// Copyright (c) 2022 Tigera Inc. All rights reserved.

package resources

import (
	"reflect"
)

// DeepEqual does reflect.DeepEqual for 2 given spec
func DeepEqual(object0 any, object1 any) bool {
	return reflect.DeepEqual(object0, object1)
}
