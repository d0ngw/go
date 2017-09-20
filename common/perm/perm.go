package perm

import (
	"context"
	"fmt"
	"strings"

	c "github.com/d0ngw/go/common"
)

// Operation 定义操作类型
type Operation int8

// 定义操作的类型
const (
	OPRead Operation = 1 << iota
	OPInsert
	OPUpdate
	OPDelete
	OPAll = OPRead | OPInsert | OPUpdate | OPDelete
)

// ParseOperation 从字符串中解析操作的权限
func ParseOperation(operation string) Operation {
	operation = strings.ToLower(operation)

	if operation == "all" {
		return OPAll
	}
	var op Operation
	for _, o := range operation {
		switch o {
		case 'r':
			op = op | OPRead
		case 'i':
			op = op | OPInsert
		case 'd':
			op = op | OPDelete
		case 'u':
			op = op | OPUpdate
		}
	}
	return op
}

// String 将权限转为字符串表达
func (p Operation) String() string {
	str := ""
	if p&OPRead != 0 {
		str += "r"
	}
	if p&OPInsert != 0 {
		str += "i"
	}
	if p&OPDelete != 0 {
		str += "d"
	}
	if p&OPUpdate != 0 {
		str += "u"
	}
	return str
}

// Resource 定义资源
type Resource struct {
	parent *Resource
	name   string
	id     string
}

// GetParent 父级资源
func (p *Resource) GetParent() *Resource {
	return p.parent
}

// GetName 资源的名称
func (p *Resource) GetName() string {
	return p.name
}

// GetID 资源的id
func (p *Resource) GetID() string {
	return p.id
}

// ResourceRegistry 记录所有的资源
type ResourceRegistry struct {
	resouceReg *c.LinkedMap
	lastError  error
}

// NewResourceRegistry 构建资源注册
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resouceReg: c.NewLinkedMap(),
		lastError:  nil,
	}
}

// Add 注册一个Resource,如果相同的资源在registry中已经存在,返回error
func (p *ResourceRegistry) Add(resource *Resource) error {
	if resource == nil {
		return fmt.Errorf("Not allow nil resource")
	}
	rid := resource.GetID()
	if _, ok := p.resouceReg.Get(rid); ok {
		return fmt.Errorf("Duplicate resouce id:%s", rid)
	}
	p.resouceReg.Put(rid, resource)
	return nil
}

// IsExist 检查指定的资源id是否存在
func (p *ResourceRegistry) IsExist(resID string) bool {
	_, ok := p.resouceReg.Get(resID)
	return ok
}

// ResourceGroup 资源分组
type ResourceGroup struct {
	Name      string      //组名称
	Resources []*Resource //资源
}

// BuildResourceGroup 构建resource group列表
func (p *ResourceRegistry) BuildResourceGroup(depth int) (groups []*ResourceGroup, err error) {
	var result = c.NewLinkedMap()
	for _, v := range p.resouceReg.Entries() {
		id := v.Key.(string)
		resource := v.Value.(*Resource)
		ids := c.SplitTrimOmitEmpty(id, ".")
		if len(ids) > depth {
			fmt.Printf("%s\n", ids)
			groupID := strings.Join(ids[0:depth], ".")
			exist, ok := result.Get(groupID)
			if !ok {
				group := &ResourceGroup{}
				groupResource, ok := p.resouceReg.Get(groupID)
				if ok && groupResource != nil {
					group.Name = groupResource.(*Resource).GetName()
				} else {
					err = fmt.Errorf("can't find group id %s", groupID)
					return
				}
				result.Put(groupID, group)
				exist = group
			}
			group := exist.(*ResourceGroup)
			group.Resources = append(group.Resources, resource)
		}
	}
	var ret []*ResourceGroup
	for _, v := range result.Entries() {
		ret = append(ret, v.Value.(*ResourceGroup))
	}
	return ret, nil
}

// NewResource 创建一个新的资源
func NewResource(name, id string, parent *Resource) *Resource {
	ids := []string{}
	if parent != nil {
		ids = append(ids, parent.GetID())
	}
	ids = append(ids, id)
	return &Resource{
		parent: parent,
		name:   name,
		id:     strings.Join(ids, "."),
	}
}

// NewResourceAndReg 创建并新建一个资源,如果相同的资源在registry中已经存在,则会panic
func NewResourceAndReg(registry *ResourceRegistry, name, id string, parent *Resource) *Resource {
	res := NewResource(name, id, parent)
	if err := registry.Add(res); err != nil {
		panic(err)
	}
	return res
}

type permKey int

const (
	required permKey = 0 //需要的权限
	user     permKey = 1 //登录的用户
)

//Perm 定义了一个权限,一个权限由资源及其对应的操作组成
type Perm struct {
	Res *Resource //资源
	Op  Operation //操作
}

// NewPerm 构建Perm
func NewPerm(res *Resource, op Operation) *Perm {
	return &Perm{Res: res, Op: op}
}

//Role 定义角色
type Role interface {
	//GetName 角色的名称
	GetName() string
	//GetPerms 角色拥有的权限
	GetPerms() map[string]Operation
}

// Principal 定义了拥有权限的主体
type Principal interface {
	// GetID 取得principal的id
	GetID() int64
	// GetName 取得principal的名称
	GetName() string
	// GetRoles 取得principal所拥有的角色
	GetRoles() []Role
}

// ReqPerm 在ctx中声明需要由perms指定的权限
func ReqPerm(ctx context.Context, perms []*Perm) (context.Context, error) {
	if ctx == nil || len(perms) == 0 {
		return ctx, fmt.Errorf("Ctx or resource must not be nil")
	}

	existed, ok := ctx.Value(required).([]*Perm)
	if ok {
		perms = append(perms, existed...)
	}

	ctx = context.WithValue(ctx, required, perms)
	return ctx, nil
}

// BindPrincipal 在ctx中绑定principal
func BindPrincipal(ctx context.Context, principal Principal) (context.Context, error) {
	if ctx == nil || principal == nil {
		return ctx, fmt.Errorf("Ctx or principal must not be nil")
	}
	ctx = context.WithValue(ctx, user, principal)
	return ctx, nil
}

// GetPrincipal 在ctx中取得principal
func GetPrincipal(ctx context.Context) (Principal, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Ctx must not be nil")
	}

	principal, ok := ctx.Value(user).(Principal)
	if ok {
		return principal, nil
	}
	return nil, nil
}

// GetRequiredPerm 在ctx中取得需要权限
func GetRequiredPerm(ctx context.Context) ([]*Perm, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Ctx must not be nil")
	}

	reqPerms, ok := ctx.Value(required).([]*Perm)
	if !ok {
		return nil, nil
	}
	return reqPerms, nil
}

// HasPermWithPrinciapl 检查principal是否拥有ctx中要求的权限
func HasPermWithPrinciapl(ctx context.Context, principal Principal) bool {
	if ctx == nil {
		return false
	}

	reqPerms, ok := ctx.Value(required).([]*Perm)
	if !ok {
		return true
	}

	return HasPermWithPrincipalAndPerms(principal, reqPerms)
}

// HasPermWithPrincipalAndPerms 检查principal是否拥有reqPerms指定的权限
func HasPermWithPrincipalAndPerms(principal Principal, reqPerms []*Perm) bool {
	if principal == nil {
		return false
	}

	if len(reqPerms) == 0 {
		return true
	}

	roles := principal.GetRoles()
	if len(roles) == 0 {
		return false
	}

	for _, r := range reqPerms {
		resID := r.Res.GetID()
		mask := r.Op
		for _, role := range roles {
			opMask, ok := role.GetPerms()[resID]
			if !ok {
				continue
			}
			mask = mask & (mask ^ opMask)
			if mask == 0 {
				break
			}
		}

		if mask != 0 {
			return false
		}
	}
	return true
}
