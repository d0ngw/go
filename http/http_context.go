package http

import (
	"context"
	"net/http"
)

type key int

const (
	errorKey key = 0 // 处理错误的key
)

// RequestWithContext 向req的context中设置key = val,返回新的request
func RequestWithContext(req *http.Request, key, val interface{}) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, key, val)
	return req.WithContext(ctx)
}

// FromRequestContext 从req的context中取得key值
func FromRequestContext(req *http.Request, key interface{}) interface{} {
	return req.Context().Value(key)
}

// RequestWithError 向req中设置当前处理的错误,返回新的request
func RequestWithError(req *http.Request, err error) *http.Request {
	return RequestWithContext(req, errorKey, err)
}

// ErrorFromRequestContext 从req的context取得错误值
func ErrorFromRequestContext(req *http.Request) (error, bool) {
	err, ok := req.Context().Value(errorKey).(error)
	return err, ok
}
