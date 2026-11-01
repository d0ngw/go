package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	c "github.com/d0ngw/go/common"
	"golang.org/x/net/netutil"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept accepts a connection and enables TCP keep-alive.
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err := tc.SetKeepAlive(true); err != nil {
		tc.Close()
		return nil, err
	}
	if err := tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		tc.Close()
		return nil, err
	}
	return tc, nil
}

// GraceableHandler tracks in-flight requests.
// Deprecated: Service now uses http.Server.Shutdown to manage connections.
type GraceableHandler struct {
	handler   http.Handler
	waitGroup *sync.WaitGroup
}

func (p *GraceableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.waitGroup.Add(1)
	defer p.waitGroup.Done()
	p.handler.ServeHTTP(w, r)
}

// Service runs an HTTP server. Configure it before initialization.
type Service struct {
	c.BaseService
	Conf      *Config
	listener  net.Listener
	serveMux  *http.ServeMux
	server    *http.Server
	serveDone chan struct{}
	stopDone  chan struct{}
	stopOK    bool
	lock      sync.Mutex
}

var _ c.Service = (*Service)(nil)
var _ c.Initable = (*Service)(nil)

// Init initializes the HTTP service and validates its handler configuration.
func (p *Service) Init() (err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.server != nil || p.stopDone != nil {
		return errors.New("HTTP service is already initialized or stopped")
	}
	if p.Conf == nil {
		return errors.New("HTTP service requires a configuration")
	}
	if p.Conf.ShutdownTimeout < 0 {
		return errors.New("HTTP shutdown timeout must not be negative")
	}
	// ServeMux panics for invalid or conflicting patterns. Return a startup error.
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("invalid HTTP handler configuration: %v", recovered)
		}
	}()
	mux := http.NewServeMux()
	for pattern, handler := range p.Conf.handles {
		if handler == nil || handler.handlerFunc == nil {
			return fmt.Errorf("nil HTTP handler for %q", pattern)
		}
		mux.Handle(pattern, p.handleWithMiddleware(handler))
	}
	addr := p.Conf.Addr
	if addr == "" {
		addr = ":http"
	}
	p.server = &http.Server{
		Addr:         addr,
		ReadTimeout:  p.Conf.ReadTimeout * time.Second,
		WriteTimeout: p.Conf.WriteTimeout * time.Second,
		Handler:      mux,
	}
	p.serveMux = mux
	return nil
}

// handleWithMiddleware wraps a handler with its middleware in order.
func (p *Service) handleWithMiddleware(handler *handlerWithMiddleware) http.HandlerFunc {
	h := MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, err := ErrorFromRequestContext(r); ok {
			c.Errorf("stop handle %s,cause by error:%s", r.RequestURI, err)
		} else {
			handler.handlerFunc(w, r)
		}
	})
	middlewares := append(append([]Middleware(nil), handler.middlewares...), p.Conf.middlewares...)
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i].Handle(h)
	}
	return http.HandlerFunc(h)
}

// Addr returns the bound address, or nil when the service is not listening.
func (p *Service) Addr() net.Addr {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.listener == nil {
		return nil
	}
	return p.listener.Addr()
}

// Start listens for connections and serves HTTP requests.
func (p *Service) Start() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.server == nil || p.listener != nil || p.stopDone != nil {
		return false
	}
	ln, err := net.Listen("tcp", p.server.Addr)
	if err != nil {
		c.Errorf("listen at %s failed: %v", p.server.Addr, err)
		return false
	}
	var listener net.Listener = tcpKeepAliveListener{ln.(*net.TCPListener)}
	if p.Conf.MaxConns > 0 {
		listener = netutil.LimitListener(listener, p.Conf.MaxConns)
	}
	server := p.server
	done := make(chan struct{})
	p.listener, p.serveDone = listener, done
	go func() {
		defer close(done)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.Errorf("HTTP server failed: %v", err)
		}
	}()
	return true
}

// Stop drains active requests up to ShutdownTimeout, then closes connections.
// It is safe to call before startup and repeatedly. A deadline failure returns false.
// Handlers must honor request cancellation; hijacked connections remain caller-owned.
func (p *Service) Stop() bool {
	p.lock.Lock()
	if p.stopDone != nil {
		done := p.stopDone
		p.lock.Unlock()
		<-done
		p.lock.Lock()
		ok := p.stopOK
		p.lock.Unlock()
		return ok
	}
	if p.server == nil {
		p.lock.Unlock()
		return true
	}
	server, serveDone := p.server, p.serveDone
	done := make(chan struct{})
	p.stopDone = done
	timeout := p.Conf.ShutdownTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	p.lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		c.Errorf("HTTP shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			c.Errorf("HTTP close failed: %v", closeErr)
		}
	}
	if serveDone != nil {
		<-serveDone
	}
	p.lock.Lock()
	p.listener = nil
	p.stopOK = err == nil
	close(done)
	p.lock.Unlock()
	return err == nil
}
