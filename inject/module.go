package inject

import (
	"fmt"
	"reflect"

	c "github.com/d0ngw/go/common"
)

// 内部的绑定实例
type internalBind struct {
	name        string            //绑定的名称
	instance    interface{}       //绑定的实例
	injectType  reflect.Type      //注入的类型
	injectValue reflect.Value     //注入的值,相当于reflect.ValueOf(instance)
	injectTags  map[string]string //用于注入的tag,用于覆盖struct field中定义的tag
}

func (p internalBind) String() string {
	return fmt.Sprintf("%v#%s", p.injectType, p.name)
}

// bindKey 用于绑定的key
type bindKey struct {
	bindName string
	bindType reflect.Type
}

func (p bindKey) String() string {
	return fmt.Sprintf("%v#%s", p.bindType, p.bindName)
}

// Provider 提供类似Guice Provider的功能,用于创建一个对象
type Provider interface {
	// GetInstance 用于创建一个实例
	GetInstance() interface{}
}

// ProviderFunc 定义用于创建一个对象的函数类型
type ProviderFunc func() interface{}

// Module 提供Guice Module的功能
type Module struct {
	binds []*internalBind
}

// NewModule 创建新的Module
func NewModule() *Module {
	return &Module{binds: []*internalBind{}}
}

// BindWithNameOverrideTags 添加带名称的绑定,injectTags用于覆盖instance中struct field中field中定义的inject tag
func (p *Module) BindWithNameOverrideTags(name string, instance interface{}, injectTags map[string]string) {
	if instance == nil {
		panic("Can't bind nil instance")
	}
	b := &internalBind{name, instance, injectType(instance), reflect.ValueOf(instance), injectTags}
	p.binds = append(p.binds, b)
}

// BindWithName 添加带名称的绑定
func (p *Module) BindWithName(name string, instance interface{}) {
	p.BindWithNameOverrideTags(name, instance, map[string]string{})
}

// Bind 添加不带名称的绑定
func (p *Module) Bind(instance interface{}) {
	p.BindWithName("", instance)
}

// BindWithProvider 通过Provider提供带名称的绑定功能
func (p *Module) BindWithProvider(name string, provider Provider) {
	if instance := provider.GetInstance(); instance != nil {
		p.BindWithName(name, instance)
		return
	}
	err := fmt.Errorf("Cant't bind nil instalce with name:%s,provider:%v", name, provider)
	panic(err)
}

// BindWithProviderFunc 通过Provider提供带名称的绑定功能
func (p *Module) BindWithProviderFunc(name string, providerFunc ProviderFunc) {
	if instance := providerFunc(); instance != nil {
		p.BindWithName(name, instance)
		return
	}
	err := fmt.Errorf("Cant't bind nil instalce with name:%s,providerFunc:%v", name, providerFunc)
	panic(err)
}

// Append 将src中的绑定追加到本module
func (p *Module) Append(src *Module) {
	if src == nil || len(src.binds) == 0 {
		return
	}
	p.binds = append(p.binds, src.binds...)
}

func checkIsInterface(typ reflect.Type) bool {
	isInterface := false
	if typ.Kind() == reflect.Ptr {
		if typ.Elem().Kind() == reflect.Interface {
			isInterface = true
		}
	} else if typ.Kind() == reflect.Interface {
		isInterface = true
	}
	return isInterface
}

// injectType 取得注入的类型,如果实例不能被注入,会抛出一个panic
func injectType(instance interface{}) reflect.Type {
	val := reflect.ValueOf(instance)
	typ := val.Type()

	//确保typ的类型不是interface{}
	if checkIsInterface(typ) {
		panic(fmt.Errorf("The type of instance `%#v` is interface,can't find it's exact type", val.Interface()))
	}
	if typ.Kind() != reflect.Ptr && reflect.Indirect(val).Kind() == reflect.Struct {
		c.Errorf("struct %T is not pointer but it's will be injected, please make sure it's expected.", instance)
	}
	return typ
}

// mergeBinds 合并多个模块的绑定,返回未命名的绑定和命名绑定
func mergeBinds(modules []*Module) (unnamed []*internalBind, named map[string][]*internalBind, all []*internalBind) {
	all = []*internalBind{}
	unnamed = []*internalBind{}
	named = map[string][]*internalBind{}

	uniqBindMap := map[bindKey]struct{}{}

	for _, module := range modules {
		for _, bind := range module.binds {
			bindkey := bindKey{bind.name, bind.injectType}
			if _, ok := uniqBindMap[bindkey]; ok {
				panic(fmt.Errorf("Duplicate bind %s", bindkey))
			} else {
				uniqBindMap[bindkey] = struct{}{}
				if len(bind.name) == 0 {
					unnamed = append(unnamed, bind)
					all = append(all, bind)
				} else {
					namedBinds := named[bind.name]
					namedBinds = append(namedBinds, bind)
					named[bind.name] = namedBinds
					all = append(all, bind)
				}
			}
		}
	}
	return
}

// Merge 合并Module
func Merge(modules ...[]*Module) []*Module {
	var merged []*Module
	for _, v := range modules {
		merged = append(merged, v...)
	}
	return merged
}
