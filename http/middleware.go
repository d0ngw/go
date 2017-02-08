package http

import (
	"net/http"
)

// MiddlewareFunc middleware函数
type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// Middleware 接口
type Middleware interface {
	// Handle处理
	Handle(next MiddlewareFunc) MiddlewareFunc
}
