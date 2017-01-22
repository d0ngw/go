//提供基本的http服务
package http

import (
	c "github.com/d0ngw/go/common"
	"golang.org/x/net/netutil"
	"net"
	"net/http"
	"sync"
	"time"
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
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

//安全地关闭的处理器
type GraceableHandler struct {
	handler   http.Handler
	waitGroup *sync.WaitGroup
}

func (self *GraceableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	self.waitGroup.Add(1)
	defer self.waitGroup.Done()

	self.handler.ServeHTTP(w, r)
}

// HttpService Http服务
type HttpService struct {
	c.BaseService
	Conf         *HttpConfig
	listener     net.Listener
	serveMux     *http.ServeMux
	graceHandler *GraceableHandler
	server       *http.Server
}

// Init 初始化Http服务
func (self *HttpService) Init() bool {
	self.Lock()
	defer self.Unlock()

	serveMux := http.NewServeMux()

	for pattern, handler := range self.Conf.handles {
		if handler == nil {
			c.Criticalf("Can't bind nil handlerFunc to path %s", pattern)
			return false
		}
		serveMux.Handle(pattern, self.handleWithMiddleware(handler))
	}

	graceHandler := &GraceableHandler{
		handler:   serveMux,
		waitGroup: &sync.WaitGroup{}}

	server := &http.Server{
		Addr:         self.Conf.Addr,
		ReadTimeout:  self.Conf.ReadTimeout * time.Second,
		WriteTimeout: self.Conf.WriteTimeout * time.Second,
		Handler:      graceHandler}

	if self.Conf.Addr == "" {
		self.Conf.Addr = ":http"
	}

	self.graceHandler = graceHandler
	self.server = server
	self.serveMux = serveMux
	return true
}

// handleWithMiddleware 依次调用各个middleware
func (self *HttpService) handleWithMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	originHandler := func(w http.ResponseWriter, r *http.Request) {
		if err, ok := ErrorFromRequestContext(r); ok {
			c.Errorf("stop handle %s,cause by error:%s", r.RequestURI, err)
		} else {
			handler(w, r)
		}
	}

	var middlewares = self.Conf.middlewares
	var middlewareCount = len(middlewares)

	h := originHandler
	for i := middlewareCount - 1; i >= 0; i-- {
		h = middlewares[i].Handle(h)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	})
}

// Start 启动Http服务,开始端口监听和服务处理
func (self *HttpService) Start() bool {
	self.Lock()
	defer self.Unlock()

	c.Infof("Listen at %s", self.Conf.Addr)
	ln, err := net.Listen("tcp", self.Conf.Addr)
	if err != nil {
		c.Errorf("Listen at %s fail,error:%v", self.Conf.Addr, err)
		return false
	}

	tcpListener := tcpKeepAliveListener{ln.(*net.TCPListener)}
	if self.Conf.MaxConns > 0 {
		self.listener = netutil.LimitListener(tcpListener, self.Conf.MaxConns)
	} else {
		self.listener = tcpListener
	}

	self.graceHandler.waitGroup.Add(1)

	go func() {
		defer self.graceHandler.waitGroup.Done()
		err := self.server.Serve(self.listener)
		if err != nil {
			c.Errorf("server.Serve return with error:%v", err)
		}
	}()
	return true
}

// Stop 停止Http服务,关闭端口监听和服务处理
func (self *HttpService) Stop() bool {
	self.Lock()
	defer self.Unlock()

	if self.listener != nil {
		if err := self.listener.Close(); err != nil {
			c.Errorf("Close listener error:%v", err)
		}
	}

	//等待所有的服务
	c.Infof("Waiting shutdown")
	self.graceHandler.waitGroup.Wait()
	c.Infof("Finish shutdown")

	self.listener = nil
	self.graceHandler = nil
	self.server = nil
	self.serveMux = nil
	return true
}
