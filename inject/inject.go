// Package inject provides Guice-style dependency injection for assembling services.
package inject

import (
	"fmt"
	"reflect"
	"strings"

	c "github.com/d0ngw/go/common"
)

// Injector assigns bound dependencies to exported, tagged struct fields.
//
//	type InjectStruct struct {
//	    Service ServiceInterface `inject:"name,optional"`
//	}
type Injector struct {
	ununamed []*internalBind
	named    map[string][]*internalBind
	all      []*internalBind
}

// Injection tag constants.
const (
	injectTagNoname = "_"
)

func (p *Injector) String() string {
	return fmt.Sprintf("Injector,unnamed binds:%v,named binds:%v", len(p.ununamed), len(p.named))
}

// NewInjector creates an injector and injects the module bindings.
func NewInjector(modules []*Module) *Injector {
	injector := &Injector{}
	injectorModule := NewModule()
	injectorModule.Bind(injector)
	modules = append(modules, injectorModule)
	unnamed, named, all := mergeBinds(modules)
	injector.ununamed = unnamed
	injector.named = named
	injector.all = all
	injector.injectModules()
	return injector
}

// Initialize initializes the bound instances.
func (p *Injector) Initialize() error {
	// Initialize bound instances.
	inits := p.GetInstancesByPrototype(struct{ s c.Initable }{})
	for _, init := range inits {
		if service, ok := init.(c.Service); ok {
			if !c.ServiceInit(service) {
				return fmt.Errorf("Init %s fail", service.Name())
			}
		} else {
			if err := init.(c.Initable).Init(); err != nil {
				return fmt.Errorf("Init %T fail,err:%s", init, err)
			}

		}
	}
	return nil
}

// Start starts the services.
func (p *Injector) Start() error {
	return p.startOrStop(true)
}

// Stop stops the services.
func (p *Injector) Stop() error {
	return p.startOrStop(false)
}

func (p *Injector) startOrStop(start bool) error {
	var services []c.Service
	for _, service := range p.GetInstancesByPrototype(struct{ s c.Service }{}) {
		services = append(services, service.(c.Service))
	}
	if len(services) == 0 {
		return nil
	}

	sortedServices := c.NewServices(services, start)
	if start {
		if !sortedServices.Start() {
			return fmt.Errorf("Start servcie fail")
		}
	} else {
		if !sortedServices.Stop() {
			return fmt.Errorf("Stop servcie fail")
		}
	}
	return nil
}

// RequireInject injects dependencies into targets.
func (p *Injector) RequireInject(targets ...interface{}) {
	for _, target := range targets {
		p.injectInstance(target)
	}
}

// RequireInjectWithOverrideTags injects dependencies using tags that override target field tags.
func (p *Injector) RequireInjectWithOverrideTags(target interface{}, injectTags map[string]string) {
	p.injectInstanceWithOverrideTags(target, injectTags)
}

// GetInstance returns the instance matching name and expectedType, or nil if none exists.
func (p *Injector) GetInstance(name string, expectedType reflect.Type) interface{} {
	bind := p.findBind(name, expectedType)
	if bind == nil {
		return nil
	}
	return bind.instance
}

// GetInstanceByPrototype returns the instance matching name and the prototype field type, or nil.
func (p *Injector) GetInstanceByPrototype(name string, prototype interface{}) interface{} {
	return p.GetInstance(name, getFieldType(prototype, 0))
}

// GetInstancesByPrototype returns all instances matching the prototype field type.
func (p *Injector) GetInstancesByPrototype(prototype interface{}) []interface{} {
	expectedType := getFieldType(prototype, 0)
	var ret = p.findAllBind(p.all, expectedType)
	var retInstances []interface{}
	for _, bind := range ret {
		retInstances = append(retInstances, bind.instance)
	}
	return retInstances
}

func (p *Injector) findAllBind(binds []*internalBind, expectedType reflect.Type) []*internalBind {
	var ret []*internalBind
	for _, bind := range binds {
		canAssign := bind.injectType.AssignableTo(expectedType)
		if canAssign {
			ret = append(ret, bind)
		}
	}
	return ret
}

// findBind finds a matching binding.
func (p *Injector) findBind(name string, expectedType reflect.Type) *internalBind {
	var toFind []*internalBind
	if len(name) != 0 {
		// Look up by name.
		if binds, ok := p.named[name]; ok {
			toFind = binds
		}
	} else {
		toFind = p.ununamed
	}

	var ret []*internalBind
	for _, bind := range toFind {
		canAssign := bind.injectType.AssignableTo(expectedType)
		if canAssign {
			ret = append(ret, bind)
		}
	}

	if len(ret) == 0 {
		return nil
	} else if len(ret) > 1 {
		bindsDesc := make([]string, 0, len(ret))
		for _, bind := range ret {
			bindsDesc = append(bindsDesc, bind.String())
		}
		panic(fmt.Errorf("Duplicate for %v#%s,available binds:[%v]", expectedType, name, strings.Join(bindsDesc, ",")))
	} else {
		return ret[0]
	}
}

// injectInstance injects dependencies into target.
func (p *Injector) injectInstance(target interface{}) {
	p.injectInstanceWithOverrideTags(target, map[string]string{})
}

// injectInstanceWithOverrideTags injects dependencies using tags that override target field tags.
func (p *Injector) injectInstanceWithOverrideTags(target interface{}, injectTags map[string]string) {
	val := reflect.ValueOf(target)
	// Only pointers to structs can receive injected dependencies.
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}
	ind := val.Elem()
	if ind.Kind() != reflect.Struct {
		return
	}
	if len(injectTags) == 0 {
		if ok, injected := IsInjected(target); ok && injected {
			return
		}
	}
	typ := ind.Type()

	fieldNum := ind.NumField()
	for i := 0; i < fieldNum; i++ {
		field := typ.Field(i)
		fieldVal := ind.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		// Inject dependencies into embedded struct fields.
		if reflect.Indirect(fieldVal).Kind() == reflect.Struct && field.Anonymous {
			if fieldVal.Kind() == reflect.Ptr {
				p.injectInstanceWithOverrideTags(fieldVal.Interface(), injectTags)
			} else if fieldVal.CanAddr() && fieldVal.CanInterface() {
				p.injectInstanceWithOverrideTags(fieldVal.Addr().Interface(), injectTags)
			}
			continue
		}

		structField := typ.Field(i)
		sfTag := structField.Tag

		tag, ok := injectTags[structField.Name]
		if !ok {
			tag = sfTag.Get("inject")
		}
		found, name, optional := parseInjectTag(tag)
		if !found {
			continue
		}

		if name == injectTagNoname {
			name = ""
		}

		if !fieldVal.CanSet() {
			panic(fmt.Errorf("Can't inject unexported  %v.%s", typ, structField.Name))
		}

		if structField.Type.Kind() == reflect.Slice {
			if name != "" {
				panic(fmt.Errorf("not support slice with name for slice %v.%s %s", typ, structField.Name, name))
			}
			elemType := structField.Type.Elem()
			var ret = p.findAllBind(p.all, elemType)
			if len(ret) == 0 && !optional {
				panic(fmt.Errorf("Can't find bind instance for %v.%s %s", typ, structField.Name, name))
			}
			var slice = reflect.MakeSlice(structField.Type, 0, len(ret))
			for _, bind := range ret {
				slice = reflect.Append(slice, bind.injectValue)
			}
			fieldVal.Set(slice)
		} else {
			foundBind := p.findBind(name, structField.Type)
			if foundBind == nil {
				if !optional {
					panic(fmt.Errorf("Can't find bind instance for %v.%s %s", typ, structField.Name, name))
				} else {
					continue
				}
			}

			if foundBind.injectValue.Kind() == reflect.Ptr && foundBind.injectValue.Pointer() == val.Pointer() {
				fieldInjectDesc := fmt.Sprintf("%s.%s", typ.Name(), structField.Name)
				injectSrcDesc := fmt.Sprintf("%#v(addr:%p)", foundBind.instance, foundBind.instance)
				injectTargetDesc := fmt.Sprintf("%s of %#v(addr:%p)", fieldInjectDesc, target, target)
				panic(fmt.Errorf("Found circular inject,src:%s,target:%s", injectSrcDesc, injectTargetDesc))
			}
			fieldVal.Set(foundBind.injectValue)
		}
	}
}

// injectModules injects dependencies into every module binding.
func (p *Injector) injectModules() {
	for _, bind := range p.ununamed {
		p.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
	}

	for _, binds := range p.named {
		for _, bind := range binds {
			p.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
		}
	}
}

// getFieldType returns the type of a struct field.
func getFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	val := reflect.Indirect(reflect.ValueOf(structObj))
	return val.Field(fieldIndex).Type()
}

// parseInjectTag parses a tag in the form "name[,optional]".
func parseInjectTag(tag string) (found bool, name string, optional bool) {
	tag = strings.TrimSpace(tag)
	if len(tag) == 0 {
		return
	}

	vals := strings.Split(tag, ",")
	if len(vals) > 0 {
		found = true
		name = strings.TrimSpace(vals[0])
	}

	others := vals[1:]
	for _, o := range others {
		o = strings.ToLower(strings.TrimSpace(o))
		if o == "optional" {
			optional = true
		}
	}
	return
}
