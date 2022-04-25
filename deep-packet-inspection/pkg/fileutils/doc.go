// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// This package has utilities to sort the the file name.
// It sorts the file according to their name such that the active file to which snort writes to `alert_fast.txt`, is
// always at the top of list, followed by all other files with name `alert_fast.txt.<unix_time>` sorted in ascending
// order of epoch time.
package fileutils
