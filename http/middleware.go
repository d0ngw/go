package http

import (
	"net/http"
	"strings"
)

// MiddlewareFunc middleware函数
type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// Middleware 接口
type Middleware interface {
	// Handle处理
	Handle(next MiddlewareFunc) MiddlewareFunc
}

// RequestMehotdMiddleware http请求方法
type RequestMehotdMiddleware struct {
	//允许的请求方法
	AllowsMethods map[string]struct{}
}

// Handle 校验Http请求的方法
func (p *RequestMehotdMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := p.AllowsMethods[r.Method]; !ok {
			http.Error(w, "Bad Request", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// NewRequestMetodMiddleware 用methods构建middleware
func NewRequestMetodMiddleware(methods ...string) *RequestMehotdMiddleware {
	m := &RequestMehotdMiddleware{
		AllowsMethods: map[string]struct{}{},
	}
	for _, method := range methods {
		m.AllowsMethods[strings.ToUpper(method)] = struct{}{}
	}
	return m
}

// Merge 合并Middleware
func Merge(middlewares ...[]Middleware) []Middleware {
	var merged []Middleware
	for _, v := range middlewares {
		merged = append(merged, v...)
	}
	return merged
}
