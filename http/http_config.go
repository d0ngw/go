// Package http 提供基本的http服务
package http

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/inject"
)

// Config Http配置
type Config struct {
	Addr          string                            //Http监听地址
	ReadTimeout   time.Duration                     //读超时,单位秒
	WriteTimeout  time.Duration                     //写超时,单位秒
	MaxConns      int                               //最大的并发连接数
	middlewares   []Middleware                      //过滤操作
	controllers   []Controller                      //controller
	handles       map[string]*handlerWithMiddleware //handles
	controllerMux sync.RWMutex
}

// NewConfig 创建配置
func NewConfig(addr string) *Config {
	return &Config{
		Addr:        addr,
		handles:     map[string]*handlerWithMiddleware{},
		middlewares: []Middleware{},
		controllers: []Controller{},
	}
}

// RegController 注册controller中的所有处理函数
func (p *Config) RegController(controller Controller) error {
	if controller == nil {
		return fmt.Errorf("Can't reg nil contriller")
	}

	p.controllers = append(p.controllers, controller)

	var path = controller.GetPath()
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

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
		if strings.HasPrefix(handlerPath, "/") {
			handlerPath = handlerPath[1:]
		}

		patternPath := path + handlerPath
		if err := p.regHandleFunc(patternPath, h); err != nil {
			return err
		}
		c.Infof("Register controller %T#%s,path:%s", controller, controller.GetName(), patternPath)
	}
	return nil
}

// RegHandleFunc 注册patternPath的处理函数handlerFunc
func (p *Config) regHandleFunc(patternPath string, handle *handlerWithMiddleware) error {
	if _, ok := p.handles[patternPath]; ok {
		return fmt.Errorf("Duplicate ,path:%s", patternPath)
	}
	p.handles[patternPath] = handle
	return nil
}

// RegHandleFunc 注册patternPath的处理函数handlerFunc
func (p *Config) RegHandleFunc(patternPath string, handlerFunc http.HandlerFunc) error {
	if _, ok := p.handles[patternPath]; ok {
		return fmt.Errorf("Duplicate ,path:%s", patternPath)
	}
	p.handles[patternPath] = &handlerWithMiddleware{handlerFunc, nil}
	return nil
}

// RegStaticFunc 注册静态资源patternPath的处理函数handlerFunc
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

// RegMiddleware 注册middleware,middleware的注册需要在RegController和RegHandleFunc之前完成
func (p *Config) RegMiddleware(middleware Middleware) error {
	if middleware == nil {
		return fmt.Errorf("invalid middleware")
	}
	p.middlewares = append(p.middlewares, middleware)
	return nil
}

// InitWithInjector 初始化操作
func (p *Config) InitWithInjector(injector *inject.Injector) error {
	for _, c := range p.controllers {
		injector.RequireInject(c)
	}
	for _, m := range p.middlewares {
		injector.RequireInject(m)
	}

	controllers := injector.GetInstancesByPrototype(struct{ s Controller }{})
	c.Infof("controller count:%d", len(controllers))
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
