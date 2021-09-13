// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package fileutils

import (
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	fileName = "alert_fast.txt"
)

// SortFile is an implementation of sort.Interface
// It sorts the files based on the unix timpestamp in the file name's postfix.
// File with name "alert_fast.txt" is considered smallest, and all other files "alert_fast.txt.<unix_time>"
// are sorted based on the <unix_time>.
type SortFile []os.FileInfo

func (f SortFile) Len() int {
	return len(f)
}

func (f SortFile) Less(i, j int) bool {
	n1 := f[i].Name()
	n2 := f[j].Name()
	if n1 == fileName {
		return true
	}
	if n2 == fileName {
		return false
	}
	t1, err := strconv.Atoi(strings.TrimPrefix(n1, fileName))
	if err != nil {
		log.WithError(err)
	}
	t2, err := strconv.Atoi(strings.TrimPrefix(n2, fileName))
	if err != nil {
		log.WithError(err)
	}
	return t1 < t2
}

func (f SortFile) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
