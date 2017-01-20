package http

import (
	"fmt"
	"net/http"
	"reflect"
	"unicode"
)

// Contoller 接口定义http处理器
type Controller interface {
	// 控制器的名称
	GetName() string
	//路径前缀,以'/'结束,同一个控制下的http.Handler都
	GetPath() string
	// Handlers返回contoller的所有处理方法,key为path,value为对应的处理方法
	GetHandlers() (map[string]http.HandlerFunc, error)
}

// BaseController 表示一个控制器
type BaseController struct {
	Name string // Controller的名称
	Path string // Controller的路径
}

func (self *BaseController) GetName() string {
	return self.Name
}

func (self *BaseController) GetPath() string {
	return self.Path
}

var (
	m http.HandlerFunc
	t = reflect.TypeOf(m)
)

// ReflectHandlers 查找controller中类型为http.HandlerFunc的可导出方法,并将驼峰命名改为下划线分隔的路径
// 例如Index -> index,GetUser -> get_user
func ReflectHandlers(controller Controller) (handlers map[string]http.HandlerFunc, err error) {
	val := reflect.ValueOf(controller)
	if !val.IsValid() || val.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("controller must be a valid pointer")
	}

	handlers = map[string]http.HandlerFunc{}
	methodCount := val.NumMethod()
	controllerType := val.Type()
	for i := 0; i < methodCount; i++ {
		methodVal := val.Method(i)
		methodValType := methodVal.Type()

		method := controllerType.Method(i)
		if methodValType.AssignableTo(t) {
			handlers[ToUnderlineName(method.Name)] = methodVal.Interface().(func(http.ResponseWriter, *http.Request))
		}
	}
	return handlers, nil
}

// ToUnderlineName 将驼峰命名改为小写的下划线命名
func ToUnderlineName(camelName string) string {
	nameRune := []rune(camelName)
	normalizeName := make([]rune, 0, len(nameRune))

	for ni := 0; ni < len(nameRune); ni++ {
		if ni != 0 && unicode.IsUpper(nameRune[ni]) && unicode.IsLower(nameRune[ni-1]) {
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
