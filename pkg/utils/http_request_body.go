// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// MIT License

// Copyright (c) 2019 Alex Edwards

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware/header"
)

// Based on: https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body

const maxBytes = 1048576 // 1MB maximum, bytes per http request body.

var (
	ErrJsonUnknownField                = errors.New("json: unknown field ")
	ErrHttpRequestBodyTooLarge         = errors.New("http: request body too large")
	ErrEmptyRequestBody                = errors.New("empty request body")
	ErrTooManyJsonObjectsInRequestBody = errors.New("To many JSON objects in request body")
)

type MalformedRequest struct {
	// Status http status code of the request error.
	Status int

	// Malformed error message.
	Msg string

	// Error cause of malformed request.
	Err error
}

// Error implementation of error type Error function, which returns the malformed request message
// as a string.
func (mr *MalformedRequest) Error() string {
	return mr.Msg
}

// decodeRequestBody decodes the json body onto a destination interface.
//
// Forms a malformed request error passes the error up to be handled.
func Decode(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &MalformedRequest{Status: http.StatusUnsupportedMediaType, Msg: msg}
		}
	}

	// Return an error if the request body is nil.
	if r.Body == nil {
		return &MalformedRequest{
			Status: http.StatusBadRequest,
			Msg:    ErrEmptyRequestBody.Error(),
			Err:    ErrEmptyRequestBody,
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg :=
				fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg, Err: syntaxError}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg, Err: io.ErrUnexpectedEOF}

		case errors.As(err, &unmarshalTypeError):
			msg :=
				fmt.Sprintf(
					"Request body contains an invalid value for the %q field (at position %d)",
					unmarshalTypeError.Field,
					unmarshalTypeError.Offset,
				)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg, Err: unmarshalTypeError}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg, Err: ErrJsonUnknownField}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg, Err: io.EOF}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &MalformedRequest{
				Status: http.StatusRequestEntityTooLarge,
				Msg:    msg,
				Err:    ErrHttpRequestBodyTooLarge,
			}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &MalformedRequest{
			Status: http.StatusBadRequest,
			Msg:    msg,
			Err:    ErrTooManyJsonObjectsInRequestBody}
	}
	return nil
}
