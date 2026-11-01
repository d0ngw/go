package http

import (
	"context"
	"net/http"
)

type key int

const (
	errorKey key = 0 // Context key for request processing errors.
)

// RequestWithContext returns a request with key and val added to its context.
func RequestWithContext(req *http.Request, key, val interface{}) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, key, val)
	return req.WithContext(ctx)
}

// FromRequestContext retrieves a value from the request context.
func FromRequestContext(req *http.Request, key interface{}) interface{} {
	return req.Context().Value(key)
}

// RequestWithError returns a request with the processing error added to its context.
func RequestWithError(req *http.Request, err error) *http.Request {
	return RequestWithContext(req, errorKey, err)
}

// ErrorFromRequestContext retrieves the processing error from the request context.
func ErrorFromRequestContext(req *http.Request) (bool, error) {
	err, ok := req.Context().Value(errorKey).(error)
	return ok, err
}
