package http

import (
	"net/http"
	"strings"
)

// MiddlewareFunc wraps an HTTP handler.
type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// Middleware defines an HTTP handler wrapper.
type Middleware interface {
	// Handle wraps the next handler.
	Handle(next MiddlewareFunc) MiddlewareFunc
}

// RequestMehotdMiddleware restricts allowed HTTP methods.
type RequestMehotdMiddleware struct {
	// Allowed HTTP methods.
	AllowsMethods map[string]struct{}
}

// Handle checks the HTTP request method.
func (p *RequestMehotdMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := p.AllowsMethods[r.Method]; !ok {
			http.Error(w, "Bad Request", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// NewRequestMetodMiddleware creates middleware for the allowed methods.
func NewRequestMetodMiddleware(methods ...string) *RequestMehotdMiddleware {
	m := &RequestMehotdMiddleware{
		AllowsMethods: map[string]struct{}{},
	}
	for _, method := range methods {
		m.AllowsMethods[strings.ToUpper(method)] = struct{}{}
	}
	return m
}

// Merge combines middleware.
func Merge(middlewares ...[]Middleware) []Middleware {
	var merged []Middleware
	for _, v := range middlewares {
		merged = append(merged, v...)
	}
	return merged
}
