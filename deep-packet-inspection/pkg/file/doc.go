// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// It ensure there are at most `maxAllowedAlertFiles` number of alert files in each directory.
// If there are more than expected number of files, it sorts the file according to epoch time in the file name,
// retains only the newest files and deletes the rest.
package file
