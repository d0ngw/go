package http

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// Controller 接口定义http处理器
type Controller interface {
	// GetName 控制器的名称
	GetName() string
	// GetPath 路径前缀,以'/'结束,同一个控制下的http.Handler都
	GetPath() string
	// GetHandlerMiddlewares 返回controller的处理方法中,需要增加middleware封装的方法,key是controller中的方法名
	GetHandlerMiddlewares() map[string][]Middleware
	// GetPatternMethods Pattern -> Controller Method
	GetPatternMethods() map[string]string
}

// BaseController 表示一个控制器
type BaseController struct {
	Name               string                  // Controller的名称
	Path               string                  // Controller的路径
	HandlerMiddlewares map[string][]Middleware // Controller中需要使用middleware封装的方法
	PatternMethods     map[string]string
}

// GetName controller的名称
func (p *BaseController) GetName() string {
	return p.Name
}

// GetPath controller的path
func (p *BaseController) GetPath() string {
	return p.Path
}

// GetHandlerMiddlewares handler的middleware
func (p *BaseController) GetHandlerMiddlewares() map[string][]Middleware {
	return p.HandlerMiddlewares
}

// GetPatternMethods return the pattern and method pairs
func (p *BaseController) GetPatternMethods() map[string]string {
	return p.PatternMethods
}

var (
	m http.HandlerFunc
	t = reflect.TypeOf(m)
)

type handlerWithMiddleware struct {
	handlerFunc http.HandlerFunc
	middlewares []Middleware
}

// ReflectHandlers 查找controller中类型为http.HandlerFunc的可导出方法,并将驼峰命名改为下划线分隔的路径
// 例如Index -> index,GetUser -> get_user
func reflectHandlers(controller Controller) (handlers map[string]*handlerWithMiddleware, err error) {
	val := reflect.ValueOf(controller)
	if !val.IsValid() || val.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("controller must be a valid pointer")
	}

	var path = controller.GetPath()

	// 检查方法是否存在
	hm := controller.GetHandlerMiddlewares()
	for name := range hm {
		if found := val.MethodByName(name); !found.IsValid() {
			return nil, fmt.Errorf("Can't find method name %s for middlewares", name)
		}
	}

	var (
		method2patterns = map[string][]string{}
	)
	for pattern, method := range controller.GetPatternMethods() {
		if found := val.MethodByName(method); !found.IsValid() {
			return nil, fmt.Errorf("Can't find method name %s for pattern ", pattern)
		}
		patterns := method2patterns[method]
		patterns = append(patterns, pattern)
		method2patterns[method] = patterns
	}

	handlers = map[string]*handlerWithMiddleware{}
	methodCount := val.NumMethod()
	controllerType := val.Type()
	for i := 0; i < methodCount; i++ {
		methodVal := val.Method(i)
		methodValType := methodVal.Type()
		method := controllerType.Method(i)

		if methodValType.AssignableTo(t) {
			var fn http.HandlerFunc = methodVal.Interface().(func(http.ResponseWriter, *http.Request))
			hmiddle := &handlerWithMiddleware{handlerFunc: fn}
			if middlewares, ok := hm[method.Name]; ok {
				hmiddle.middlewares = middlewares
			}

			patterns := method2patterns[method.Name]
			if len(patterns) > 0 {
				for _, pattern := range patterns {
					subs := strings.SplitN(pattern, "/", 2)
					if len(subs) != 2 {
						return nil, fmt.Errorf("invalid pattern: %s", pattern)
					}
					if subs[1] != "" && path != "/" {
						subs[1] = "/" + subs[1]
					}
					pattern = subs[0] + path + subs[1]
					handlers[pattern] = hmiddle

				}
			} else {
				var handlerPath = path
				if path != "/" {
					handlerPath += "/"
				}
				handlerPath += ToUnderlineName(method.Name)
				handlers[handlerPath] = hmiddle
			}
		}
	}
	return handlers, nil
}

// ToUnderlineName 将驼峰命名改为小写的下划线命名
func ToUnderlineName(camelName string) string {
	nameRune := []rune(camelName)
	normalizeName := make([]rune, 0, len(nameRune))

	for ni := 0; ni < len(nameRune); ni++ {
		if ni != 0 && unicode.IsUpper(nameRune[ni]) && !unicode.IsUpper(nameRune[ni-1]) {
			normalizeName = append(normalizeName, '_')
		}

		r := nameRune[ni]
		if unicode.IsUpper(nameRune[ni]) {
			r = unicode.ToLower(r)
		}
		normalizeName = append(normalizeName, r)
	}
	return string(normalizeName)
}
