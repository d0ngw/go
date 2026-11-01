// Package http provides HTTP services, controllers, and client helpers.
package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/inject"
)

// Config configures an HTTP service.
type Config struct {
	Addr            string                            // HTTP listen address.
	ReadTimeout     time.Duration                     // Read timeout in seconds.
	WriteTimeout    time.Duration                     // Write timeout in seconds.
	ShutdownTimeout time.Duration                     // Graceful shutdown duration; zero defaults to 30 seconds.
	MaxConns        int                               // Maximum number of concurrent connections.
	middlewares     []Middleware                      // Request middleware.
	controllers     []Controller                      //controller
	handles         map[string]*handlerWithMiddleware //handles
	controllerMux   sync.RWMutex
}

// NewConfig creates an HTTP service configuration.
func NewConfig(addr string) *Config {
	return &Config{
		Addr:        addr,
		handles:     map[string]*handlerWithMiddleware{},
		middlewares: []Middleware{},
		controllers: []Controller{},
	}
}

// RegController registers all handler methods on controller.
func (p *Config) RegController(controller Controller) error {
	if controller == nil {
		return fmt.Errorf("Can't reg nil contriller")
	}

	p.controllers = append(p.controllers, controller)

	p.controllerMux.Lock()
	defer p.controllerMux.Unlock()

	handlers, err := reflectHandlers(controller)
	if err != nil {
		return err
	}

	if len(handlers) == 0 {
		c.Warnf("Can't find handler in %#v", controller)
		return nil
	}

	for handlerPath, h := range handlers {
		if err := p.regHandleFunc(handlerPath, h); err != nil {
			return err
		}
	}
	return nil
}

// RegHandleFunc registers handlerFunc for patternPath.
func (p *Config) regHandleFunc(patternPath string, handle *handlerWithMiddleware) error {
	if _, ok := p.handles[patternPath]; ok {
		return fmt.Errorf("Duplicate ,path:%s", patternPath)
	}
	p.handles[patternPath] = handle
	return nil
}

// RegHandleFunc registers handlerFunc for patternPath.
func (p *Config) RegHandleFunc(patternPath string, handlerFunc http.HandlerFunc) error {
	if _, ok := p.handles[patternPath]; ok {
		return fmt.Errorf("Duplicate ,path:%s", patternPath)
	}
	p.handles[patternPath] = &handlerWithMiddleware{handlerFunc, nil}
	return nil
}

// RegStaticFunc registers handlers for static resource paths.
func (p *Config) RegStaticFunc(patternAndPath map[string]string) error {
	if patternAndPath == nil {
		return nil
	}
	for pattern, path := range patternAndPath {
		httpDir := http.Dir(path)
		fs := http.FileServer(NoDirFS{Fs: httpDir})
		c.Infof("add static %s to %s", pattern, httpDir)
		err := p.RegHandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// RegMiddleware registers middleware before controllers and handlers are registered.
func (p *Config) RegMiddleware(middleware Middleware) error {
	if middleware == nil {
		return fmt.Errorf("invalid middleware")
	}
	p.middlewares = append(p.middlewares, middleware)
	return nil
}

// InitWithInjector injects dependencies and registers controllers.
func (p *Config) InitWithInjector(injector *inject.Injector) error {
	for _, c := range p.controllers {
		injector.RequireInject(c)
	}
	for _, m := range p.middlewares {
		injector.RequireInject(m)
	}

	controllers := injector.GetInstancesByPrototype(struct{ s Controller }{})
	for _, cer := range controllers {
		controller := cer.(Controller)
		if err := p.RegController(controller); err != nil {
			return fmt.Errorf("Reg controller %s fail,err:%s", controller.GetName(), err)
		}
		for _, middlewares := range controller.GetHandlerMiddlewares() {
			for _, m := range middlewares {
				injector.RequireInject(m)
			}
		}
	}
	return nil
}
