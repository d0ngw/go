package http

import (
	c "github.com/d0ngw/go/common"
	"net/http"
)

type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// HttpMiddleware定义接口
type HttpMiddleware interface {
	// Handle处理
	Handle(next MiddlewareFunc) MiddlewareFunc
}

type LogMiddleware struct {
	Order int
}

func NewLogMiddleware(order int) *LogMiddleware {
	return &LogMiddleware{
		Order: order,
	}
}

func (p *LogMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.Infof("Begin process:%s,order:%d", r.RequestURI, p.Order)
		next(w, r)
		c.Infof("Finish process:%s,order:%d", r.RequestURI, p.Order)
	}
}
