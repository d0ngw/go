package http

import (
	"net/http"
)

type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// HttpMiddleware定义接口
type HttpMiddleware interface {
	// Handle处理
	Handle(next MiddlewareFunc) MiddlewareFunc
}
