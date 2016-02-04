// inject 提供类似Guice Injector的依赖注入功能,可以用于简化服务的组装
package inject

import (
	"fmt"
	"log"
	"reflect"
	"strings"
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
	inject_tag_noname = "_" //无名称注入
)

func (self *Injector) String() string {
	return fmt.Sprintf("Injector,unnamed binds:%v,named binds:%v", len(self.ununamed), len(self.named))
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
func (self *Injector) RequireInject(targets ...interface{}) {
	for _, target := range targets {
		self.injectInstance(target)
	}
}

// RequireInjectWithOverrideTags 由Injector向target注入服务,injectTags用于覆盖target struct字段中定义的inject tag
func (self *Injector) RequireInjectWithOverrideTags(target interface{}, injectTags map[string]string) {
	self.injectInstanceWithOverrideTags(target, injectTags)
}

// GetInstance 从Injector中查找与name和expectedType匹配的实例,如果没有找到返回nil
func (self *Injector) GetInstance(name string, expectedType reflect.Type) interface{} {
	bind := self.findBind(name, expectedType)
	if bind == nil {
		return nil
	} else {
		return bind.instance
	}
}

// GetInstanceByKey 从Injector重查找与name和prototype匹配的实例,如果没有找到返回nil
func (self *Injector) GetInstanceByPrototype(name string, prototype interface{}) interface{} {
	return self.GetInstance(name, getFieldType(prototype, 0))
}

// GetInstancesByPrototype 从Injector里查找与prototype匹配的所有实例
func (self *Injector) GetInstancesByPrototype(prototype interface{}) []interface{} {
	expectedType := getFieldType(prototype, 0)
	var ret []*internalBind = self.findAllBind(self.all, expectedType)
	var retInstances []interface{}
	for _, bind := range ret {
		retInstances = append(retInstances, bind.instance)
	}
	return retInstances
}

func (self *Injector) findAllBind(binds []*internalBind, expectedType reflect.Type) []*internalBind {
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
func (self *Injector) findBind(name string, expectedType reflect.Type) *internalBind {
	var toFind []*internalBind
	if len(name) != 0 {
		//按名称查找
		if binds, ok := self.named[name]; ok {
			toFind = binds
		}
	} else {
		toFind = self.ununamed
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
func (self *Injector) injectInstance(target interface{}) {
	self.injectInstanceWithOverrideTags(target, map[string]string{})
}

// injectInstanceWithOverrideTAgs 向target注入实例,injectTags用于覆盖target struct中定义的inject tag定义
func (self *Injector) injectInstanceWithOverrideTags(target interface{}, injectTags map[string]string) {
	val := reflect.ValueOf(target)
	ind := reflect.Indirect(val)
	typ := ind.Type()

	//仅struct的指针才能被注入
	if val.Kind() != reflect.Ptr || ind.Kind() != reflect.Struct {
		debugf("Skip inject:%#v", target)
		return
	}

	fieldNum := ind.NumField()
	for i := 0; i < fieldNum; i++ {
		fieldVal := ind.Field(i)
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

		if name == inject_tag_noname {
			name = ""
		}

		foundBind := self.findBind(name, structField.Type)
		if foundBind == nil && !optional {
			panic(fmt.Errorf("Can't find bind instance for %v.%s", typ, structField.Name))
		}

		if !fieldVal.CanSet() {
			panic(fmt.Errorf("Can't inject unexported  %v.%s", typ, structField.Name))
		}

		fieldInjectDesc := fmt.Sprintf("%s.%s", typ.Name(), structField.Name)
		debugf("Inject %#v(kind:%v) to %s", foundBind.instance, foundBind.injectValue.Kind(), fieldInjectDesc)
		if foundBind.injectValue.Kind() == reflect.Ptr && foundBind.injectValue.Pointer() == val.Pointer() {
			injectSrcDesc := fmt.Sprintf("%#v(addr:%p)", foundBind.instance, foundBind.instance)
			injectTargetDesc := fmt.Sprintf("%s of %#v(addr:%p)", fieldInjectDesc, target, target)
			panic(fmt.Errorf("Found circular inject,src:%s,target:%s", injectSrcDesc, injectTargetDesc))
		}
		fieldVal.Set(foundBind.injectValue)
	}
}

// injectModules 注入Injector中各个Module中的绑定
func (self *Injector) injectModules() {
	debugf("Inject internal")
	for _, bind := range self.ununamed {
		self.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
	}

	for _, binds := range self.named {
		for _, bind := range binds {
			self.injectInstanceWithOverrideTags(bind.instance, bind.injectTags)
		}
	}
}

// debugf 输出日志
func debugf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// getFieldType 取得struct中field的类型
func getFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	val := reflect.Indirect(reflect.ValueOf(structObj))
	return val.Field(fieldIndex).Type()
}
