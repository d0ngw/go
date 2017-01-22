//提供基本的http服务
package http

import (
	"fmt"
	c "github.com/d0ngw/go/common"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HttpConfig Http配置
type HttpConfig struct {
	Addr          string                      //Http监听地址
	ReadTimeout   time.Duration               //读超时,单位秒
	WriteTimeout  time.Duration               //写超时,单位秒
	MaxConns      int                         //最大的并发连接数
	middlewares   []HttpMiddleware            //过滤操作
	handles       map[string]http.HandlerFunc //Controller配置,key:uri pattern,value:http.Handler 的名称
	controllerMux sync.RWMutex
}

func NewHttpConfig(addr string) *HttpConfig {
	return &HttpConfig{
		Addr:        addr,
		handles:     map[string]http.HandlerFunc{},
		middlewares: []HttpMiddleware{},
	}
}

// RegController 注册controller中的所有处理函数
func (p *HttpConfig) RegController(controller Controller) error {
	if controller == nil {
		return fmt.Errorf("Can't reg nil contriller")
	}

	c.Infof("Reg controller %s", controller.GetName())
	var path = controller.GetPath()
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	p.controllerMux.Lock()
	defer p.controllerMux.Unlock()

	handlers, err := controller.GetHandlers()
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
		if err := p.RegHandleFunc(patternPath, h); err != nil {
			return err
		} else {
			c.Infof("Register controller %s,path:%s", controller.GetName(), patternPath)
		}
	}
	return nil
}

// RegHandleFunc 注册patternPath的处理函数handlerFunc
func (p *HttpConfig) RegHandleFunc(patternPath string, handlerFunc http.HandlerFunc) error {
	if _, ok := p.handles[patternPath]; ok {
		return fmt.Errorf("Duplicate ,path:%s", patternPath)
	} else {
		p.handles[patternPath] = handlerFunc
	}
	return nil
}

// RegMiddleware 注册middleware,middleware的注册需要在RegController和RegHandleFunc之前完成
func (p *HttpConfig) RegMiddleware(middleware HttpMiddleware) error {
	if middleware == nil {
		return fmt.Errorf("invalid middleware")
	}
	p.middlewares = append(p.middlewares, middleware)
	return nil
}
