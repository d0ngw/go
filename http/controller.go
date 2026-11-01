package http

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// Controller defines a group of HTTP handlers.
type Controller interface {
	// GetName returns the controller name.
	GetName() string
	// GetPath returns the path prefix shared by the controller's handlers, ending in '/'.
	GetPath() string
	// GetHandlerMiddlewares returns middleware keyed by controller method name.
	GetHandlerMiddlewares() map[string][]Middleware
	// GetPatternMethods Pattern -> Controller Method
	GetPatternMethods() map[string]string
}

// BaseController provides a basic controller implementation.
type BaseController struct {
	Name               string                  // Controller name.
	Path               string                  // Controller path prefix.
	HandlerMiddlewares map[string][]Middleware // Middleware keyed by controller method name.
	PatternMethods     map[string]string
}

// GetName returns the controller name.
func (p *BaseController) GetName() string {
	return p.Name
}

// GetPath returns the controller path prefix.
func (p *BaseController) GetPath() string {
	return p.Path
}

// GetHandlerMiddlewares returns the middleware for each handler.
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

// reflectHandlers finds exported HTTP handler methods and derives paths using snake_case names.
// For example, Index becomes index and GetUser becomes get_user.
func reflectHandlers(controller Controller) (handlers map[string]*handlerWithMiddleware, err error) {
	val := reflect.ValueOf(controller)
	if !val.IsValid() || val.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("controller must be a valid pointer")
	}

	var path = controller.GetPath()

	// Check that the method exists.
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

// ToUnderlineName converts a camel-case name to lowercase snake_case.
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
