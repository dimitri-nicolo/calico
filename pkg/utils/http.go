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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware/header"
	log "github.com/sirupsen/logrus"
)

// Based on: https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body

const maxBytes = 1048576 // 1MB maximum, bytes per http request body.

var (
	ErrJsonUnknownField                = errors.New("json: unknown field ")
	ErrHttpRequestBodyTooLarge         = errors.New("http: request body too large")
	ErrCantReadRequestBody             = errors.New("can't read body")
	ErrEmptyRequestBody                = errors.New("empty request body")
	ErrTooManyJsonObjectsInRequestBody = errors.New("too many JSON objects in request body")
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

// Decode decodes the json body onto a destination interface.
//
// Decodes and maintains the request body onto the next handler. Forms a
// malformed request error passes the error up to be handled.
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

	// Limit the allowable request body size.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	// Retain the body, to pass it forward.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if len(body) >= maxBytes {
			msg := "Request body must not be larger than 1MB"
			return &MalformedRequest{
				Status: http.StatusRequestEntityTooLarge,
				Msg:    msg,
				Err:    ErrHttpRequestBodyTooLarge,
			}
		} else {
			msg := "Cannot read request body"
			return &MalformedRequest{
				Status: http.StatusBadRequest,
				Msg:    msg,
				Err:    ErrCantReadRequestBody,
			}
		}
	}

	dec := json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&dst); err != nil {
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

		default:
			return err
		}
	}

	// Limit to one srtuct per request.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &MalformedRequest{
			Status: http.StatusBadRequest,
			Msg:    msg,
			Err:    ErrTooManyJsonObjectsInRequestBody}
	}

	// Write data back to the request body.
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return nil
}

// Encode encodes the src as a JSON response to the responce writer destination.
func Encode(dst http.ResponseWriter, src interface{}) error {
	dst.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(dst).Encode(src); err != nil {
		msg := "Encoding search results failed"
		log.Debug(msg)
		return &MalformedRequest{
			Status: http.StatusBadRequest,
			Msg:    msg,
			Err:    ErrTooManyJsonObjectsInRequestBody}
	}

	return nil
}
