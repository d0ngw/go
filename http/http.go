package http

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	c "github.com/d0ngw/go/common"
	"golang.org/x/net/netutil"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept接受连接
func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err = tc.SetKeepAlive(true); err != nil {
		return
	}
	if err = tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		return
	}
	return tc, nil
}

// GraceableHandler 安全地关闭的处理器
type GraceableHandler struct {
	handler   http.Handler
	waitGroup *sync.WaitGroup
}

func (p *GraceableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.waitGroup.Add(1)
	defer p.waitGroup.Done()

	p.handler.ServeHTTP(w, r)
}

// Service Http服务
type Service struct {
	c.BaseService
	Conf         *Config
	listener     net.Listener
	serveMux     *http.ServeMux
	graceHandler *GraceableHandler
	server       *http.Server
	lock         sync.Mutex
}

// Init 初始化Http服务
func (p *Service) Init() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	serveMux := http.NewServeMux()

	for pattern, handler := range p.Conf.handles {
		if handler == nil {
			c.Errorf("Can't bind nil handlerFunc to path %s", pattern)
			return false
		}
		serveMux.Handle(pattern, p.handleWithMiddleware(handler))
	}

	graceHandler := &GraceableHandler{
		handler:   serveMux,
		waitGroup: &sync.WaitGroup{}}

	server := &http.Server{
		Addr:         p.Conf.Addr,
		ReadTimeout:  p.Conf.ReadTimeout * time.Second,
		WriteTimeout: p.Conf.WriteTimeout * time.Second,
		Handler:      graceHandler}

	if p.Conf.Addr == "" {
		p.Conf.Addr = ":http"
	}

	p.graceHandler = graceHandler
	p.server = server
	p.serveMux = serveMux
	return true
}

// handleWithMiddleware 依次调用各个middleware
func (p *Service) handleWithMiddleware(handler *handlerWithMiddleware) http.HandlerFunc {
	originHandler := func(w http.ResponseWriter, r *http.Request) {
		if ok, err := ErrorFromRequestContext(r); ok {
			c.Errorf("stop handle %s,cause by error:%s", r.RequestURI, err)
		} else {
			handler.handlerFunc(w, r)
		}
	}

	var middlewares = append(handler.middlewares, p.Conf.middlewares...)
	var middlewareCount = len(middlewares)

	h := originHandler
	for i := middlewareCount - 1; i >= 0; i-- {
		m := middlewares[i]
		h0 := h
		h = func(w http.ResponseWriter, r *http.Request) {
			m.Handle(h0)(w, r)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	})
}

// Start 启动Http服务,开始端口监听和服务处理
func (p *Service) Start() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	c.Infof("Listen at %s", p.Conf.Addr)
	ln, err := net.Listen("tcp", p.Conf.Addr)
	if err != nil {
		c.Errorf("Listen at %s fail,error:%v", p.Conf.Addr, err)
		return false
	}

	tcpListener := tcpKeepAliveListener{ln.(*net.TCPListener)}
	if p.Conf.MaxConns > 0 {
		p.listener = netutil.LimitListener(tcpListener, p.Conf.MaxConns)
	} else {
		p.listener = tcpListener
	}

	p.graceHandler.waitGroup.Add(1)

	go func() {
		defer p.graceHandler.waitGroup.Done()
		err := p.server.Serve(p.listener)
		if err != nil {
			var errLevel = c.Error
			if strings.Contains(err.Error(), "use of closed network connection") {
				errLevel = c.Warn
			}
			c.Logf(errLevel, "server.Serve return with %v", err)
		}
	}()
	return true
}

// Stop 停止Http服务,关闭端口监听和服务处理
func (p *Service) Stop() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.listener != nil {
		if err := p.listener.Close(); err != nil {
			c.Errorf("Close listener error:%v", err)
		}
	}

	//等待所有的服务
	c.Infof("Waiting shutdown")
	p.graceHandler.waitGroup.Wait()
	c.Infof("Finish shutdown")

	p.listener = nil
	p.graceHandler = nil
	p.server = nil
	p.serveMux = nil
	return true
}
