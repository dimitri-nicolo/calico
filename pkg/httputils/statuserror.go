// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package httputils

type HttpStatusError struct {
	// Status http status code of the request error.
	Status int

	// Http status error message.
	Msg string

	// Error cause of http status request.
	Err error
}

// Error implementation of error type Error function, which returns the http status message
// as a string.
func (mr *HttpStatusError) Error() string {
	return mr.Msg
}
