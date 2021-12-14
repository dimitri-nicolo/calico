// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package strutil

func InList(str string, list []string) bool {
	for _, entry := range list {
		if entry == str {
			return true
		}
	}

	return false
}
