// Package inject 提供类似Guice Injector的依赖注入功能,可以用于简化服务的组装
package inject

import (
	"fmt"
	"reflect"
	"strings"

	c "github.com/d0ngw/go/common"
)

// Injector 类似Guice的注入器
// 绑定约束:相同类型的实例和名称在一个Injector中必须唯一
// Injector对于struct中可导出的字段进行注入,需要注入的字段应该使用inject tag进行说明
//  type InjectStruct struct{
//   s ServiceInterface{} `inject:"<name>,[optional]"`
//  }
// inject格式:
//  name: 绑定的名称,必须的,如果没有名称,使用`_`代替
//  optional: 标识可选的注入,即不是必须注入
type Injector struct {
	ununamed []*internalBind            //未命名的绑定
	named    map[string][]*internalBind //命名的绑定
	all      []*internalBind            //所有的绑定
}

// Inject tag常量
const (
	injectTagNoname = "_" //无名称注入
)

func (p *Injector) String() string {
	return fmt.Sprintf("Injector,unnamed binds:%v,named binds:%v", len(p.ununamed), len(p.named))
}

// parseInjectTag 解析inject tag,inject tag的格式
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

// NewInjector 创建一个Injector
func NewInjector(modules []*Module) *Injector {
	unnamed, named, all := mergeBinds(modules)
	injector := &Injector{unnamed, named, all}
	injector.injectModules()
	return injector
}

// RequireInject 由Injector向targets注入服务
func (p *Injector) RequireInject(targets ...interface{}) {
	for _, target := range targets {
		p.injectInstance(target)
	}
}

// RequireInjectWithOverrideTags 由Injector向target注入服务,injectTags用于覆盖target struct字段中定义的inject tag
func (p *Injector) RequireInjectWithOverrideTags(target interface{}, injectTags map[string]string) {
	p.injectInstanceWithOverrideTags(target, injectTags)
}

// GetInstance 从Injector中查找与name和expectedType匹配的实例,如果没有找到返回nil
func (p *Injector) GetInstance(name string, expectedType reflect.Type) interface{} {
	bind := p.findBind(name, expectedType)
	if bind == nil {
		return nil
	}
	return bind.instance
}

// GetInstanceByPrototype 从Injector重查找与name和prototype匹配的实例,如果没有找到返回nil
func (p *Injector) GetInstanceByPrototype(name string, prototype interface{}) interface{} {
	return p.GetInstance(name, getFieldType(prototype, 0))
}

// GetInstancesByPrototype 从Injector里查找与prototype匹配的所有实例
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

// findBind 寻找一个可用的绑定
func (p *Injector) findBind(name string, expectedType reflect.Type) *internalBind {
	var toFind []*internalBind
	if len(name) != 0 {
		//按名称查找
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

// injectInstance 向target注入实例
func (p *Injector) injectInstance(target interface{}) {
	p.injectInstanceWithOverrideTags(target, map[string]string{})
}

// injectInstanceWithOverrideTAgs 向target注入实例,injectTags用于覆盖target struct中定义的inject tag定义
func (p *Injector) injectInstanceWithOverrideTags(target interface{}, injectTags map[string]string) {
	val := reflect.ValueOf(target)
	ind := reflect.Indirect(val)
	typ := ind.Type()

	//仅struct的指针才能被注入
	if val.Kind() != reflect.Ptr || ind.Kind() != reflect.Struct {
		c.Debugf("Skip inject:%#v", target)
		return
	}

	fieldNum := ind.NumField()
	for i := 0; i < fieldNum; i++ {
		field := typ.Field(i)
		fieldVal := ind.Field(i)
		if !fieldVal.CanSet() {
			c.Debugf("skip inject filed %s", field.Name)
			continue
		}

		if reflect.Indirect(fieldVal).Kind() == reflect.Struct {
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

		foundBind := p.findBind(name, structField.Type)
		if foundBind == nil {
			if !optional {
				panic(fmt.Errorf("Can't find bind instance for %v.%s", typ, structField.Name))
			} else {
				c.Infof("skip optional %s %s ", name, structField.Type)
				continue
			}
		}

		if !fieldVal.CanSet() {
			panic(fmt.Errorf("Can't inject unexported  %v.%s", typ, structField.Name))
		}

		fieldInjectDesc := fmt.Sprintf("%s.%s", typ.Name(), structField.Name)
		c.Debugf("Inject %T#%s to %s", foundBind.instance, foundBind.name, fieldInjectDesc)
		if foundBind.injectValue.Kind() == reflect.Ptr && foundBind.injectValue.Pointer() == val.Pointer() {
			injectSrcDesc := fmt.Sprintf("%#v(addr:%p)", foundBind.instance, foundBind.instance)
			injectTargetDesc := fmt.Sprintf("%s of %#v(addr:%p)", fieldInjectDesc, target, target)
			panic(fmt.Errorf("Found circular inject,src:%s,target:%s", injectSrcDesc, injectTargetDesc))
		}
		fieldVal.Set(foundBind.injectValue)
	}
}

// injectModules 注入Injector中各个Module中的绑定
func (p *Injector) injectModules() {
	c.Debugf("Inject internal")
	for _, bind := range p.ununamed {
		p.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
	}

	for _, binds := range p.named {
		for _, bind := range binds {
			p.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
		}
	}
}

// getFieldType 取得struct中field的类型
func getFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	val := reflect.Indirect(reflect.ValueOf(structObj))
	return val.Field(fieldIndex).Type()
}

// Merge 合并Module
func Merge(modules ...[]*Module) []*Module {
	var merged []*Module
	for _, v := range modules {
		merged = append(merged, v...)
	}
	return merged
}
