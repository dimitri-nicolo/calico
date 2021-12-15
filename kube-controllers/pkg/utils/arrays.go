// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package utils

func DeleteFromArray(arr []string, value string) []string {
	for i, item := range arr {
		if item == value {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}
