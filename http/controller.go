package http

import (
	"fmt"
	c "github.com/d0ngw/go/common"
	"net/http"
	"reflect"
	"sync"
)

//控制器
type Controller struct {
	Name string //Con
}

//默认实现,返回501
func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

//controller注册
var controllerMux sync.RWMutex
var controllers = make(map[string]http.Handler)

// RegController 根据名称注册一个controller
func RegController(name string, controller http.Handler) {
	if len(name) == 0 {
		panic("The controller name must not be empty")
	}
	if reflect.ValueOf(controller).Kind() != reflect.Ptr {
		panic("The controller must be a pointer")
	}

	controllerMux.Lock()
	defer controllerMux.Unlock()

	if _, ok := controllers[name]; ok {
		panic(fmt.Errorf("Duplicate controller name:%s", name))
	} else {
		controllers[name] = controller
		c.Infof("Register controller name:%s with %T", controller)
	}
}

// GetController 根据名称查找Controller
func GetController(name string) http.Handler {
	if len(name) == 0 {
		panic("The controller name must not be empty")
	}

	controllerMux.RLock()
	defer controllerMux.RUnlock()

	if v, ok := controllers[name]; ok {
		return v
	}
	return nil
}
