// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package httputils

import (
	"errors"
	"net/http"
)

type HttpStatusError struct {
	// Status http status code of the request error.
	Status int `json:"status"`

	// Http status error message.
	Msg string `json:"msg"`

	// Error cause of http status request.
	Err error `json:"-"`
}

// Error implementation of error type Error function, which returns the http status message
// as a string.
func (mr *HttpStatusError) Error() string {
	return mr.Msg
}

func NewHttpStatusErrorBadRequest(msg string, err error) error {
	if err == nil {
		err = errors.New(msg)
	}
	return &HttpStatusError{
		Status: http.StatusBadRequest,
		Msg:    msg,
		Err:    err,
	}
}
