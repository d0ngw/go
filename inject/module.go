package inject

import (
	"fmt"
	"reflect"

	c "github.com/d0ngw/go/common"
)

// Internal representation of a binding.
type internalBind struct {
	name        string            // Binding name.
	instance    interface{}       // Bound instance.
	injectType  reflect.Type      // Type of the injected instance.
	injectValue reflect.Value     // Injected value, equivalent to reflect.ValueOf(instance).
	injectTags  map[string]string // Injection tags that override struct field tags.
}

func (p internalBind) String() string {
	return fmt.Sprintf("%v#%s", p.injectType, p.name)
}

// bindKey identifies a binding.
type bindKey struct {
	bindName string
	bindType reflect.Type
}

func (p bindKey) String() string {
	return fmt.Sprintf("%v#%s", p.bindType, p.bindName)
}

// Provider creates instances, like a Guice provider.
type Provider interface {
	// GetInstance creates an instance.
	GetInstance() interface{}
}

// ProviderFunc is a function that creates an instance.
type ProviderFunc func() interface{}

// Module groups dependency bindings, like a Guice module.
type Module struct {
	binds []*internalBind
}

// NewModule creates a module.
func NewModule() *Module {
	return &Module{binds: []*internalBind{}}
}

// BindWithNameOverrideTags adds a named binding with injection tags that override instance field tags.
func (p *Module) BindWithNameOverrideTags(name string, instance interface{}, injectTags map[string]string) {
	if instance == nil {
		panic("Can't bind nil instance")
	}
	b := &internalBind{name, instance, injectType(instance), reflect.ValueOf(instance), injectTags}
	p.binds = append(p.binds, b)
}

// BindWithName adds a named binding.
func (p *Module) BindWithName(name string, instance interface{}) {
	p.BindWithNameOverrideTags(name, instance, map[string]string{})
}

// Bind adds an unnamed binding.
func (p *Module) Bind(instance interface{}) {
	p.BindWithName("", instance)
}

// BindWithProvider adds a named binding using a provider.
func (p *Module) BindWithProvider(name string, provider Provider) {
	if instance := provider.GetInstance(); instance != nil {
		p.BindWithName(name, instance)
		return
	}
	err := fmt.Errorf("Cant't bind nil instalce with name:%s,provider:%v", name, provider)
	panic(err)
}

// BindWithProviderFunc adds a named binding using a provider function.
func (p *Module) BindWithProviderFunc(name string, providerFunc ProviderFunc) {
	if instance := providerFunc(); instance != nil {
		p.BindWithName(name, instance)
		return
	}
	err := fmt.Errorf("Cant't bind nil instalce with name:%s,providerFunc:%v", name, providerFunc)
	panic(err)
}

// Append adds the bindings from src to this module.
func (p *Module) Append(src *Module) {
	if src == nil || len(src.binds) == 0 {
		return
	}
	p.binds = append(p.binds, src.binds...)
}

// IsBind check if the instance has been binded
func (p *Module) IsBind(instance interface{}, name string) bool {
	for _, internal := range p.binds {
		if internal.instance == instance && internal.name == name {
			return true
		}
	}
	return false
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

// injectType returns the injectable type and panics if the instance cannot be injected.
func injectType(instance interface{}) reflect.Type {
	val := reflect.ValueOf(instance)
	typ := val.Type()

	// Ensure typ is not an empty interface.
	if checkIsInterface(typ) {
		panic(fmt.Errorf("The type of instance `%#v` is interface,can't find it's exact type", val.Interface()))
	}
	if typ.Kind() != reflect.Ptr && reflect.Indirect(val).Kind() == reflect.Struct {
		c.Errorf("struct %T is not pointer but it's will be injected, please make sure it's expected.", instance)
	}
	return typ
}

// mergeBinds combines modules and returns unnamed, named, and all bindings.
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

// Merge combines modules.
func Merge(modules ...[]*Module) []*Module {
	var merged []*Module
	for _, v := range modules {
		merged = append(merged, v...)
	}
	return merged
}
