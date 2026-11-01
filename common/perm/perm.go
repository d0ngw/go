package perm

import (
	"context"
	"fmt"
	"strings"

	c "github.com/d0ngw/go/common"
)

// Operation represents permission operations.
type Operation int8

// Supported permission operations.
const (
	OPRead Operation = 1 << iota
	OPInsert
	OPUpdate
	OPDelete
	OPAll = OPRead | OPInsert | OPUpdate | OPDelete
)

// ParseOperation parses permission operations from a string.
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

// String returns the permission operations as a string.
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

// Resource represents a protected resource.
type Resource struct {
	parent *Resource
	name   string
	id     string
}

// GetParent returns the parent resource.
func (p *Resource) GetParent() *Resource {
	return p.parent
}

// GetName returns the resource name.
func (p *Resource) GetName() string {
	return p.name
}

// GetID returns the resource ID.
func (p *Resource) GetID() string {
	return p.id
}

// ResourceRegistry stores registered resources.
type ResourceRegistry struct {
	resouceReg *c.LinkedMap
	lastError  error
}

// NewResourceRegistry creates a resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resouceReg: c.NewLinkedMap(),
		lastError:  nil,
	}
}

// Add registers a resource and returns an error if its ID already exists.
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

// IsExist reports whether the resource ID is registered.
func (p *ResourceRegistry) IsExist(resID string) bool {
	_, ok := p.resouceReg.Get(resID)
	return ok
}

// ResourceGroup groups related resources.
type ResourceGroup struct {
	Name      string      // Group name.
	Resources []*Resource // Resources.
}

// BuildResourceGroup builds a list of resource groups.
func (p *ResourceRegistry) BuildResourceGroup(depth int) (groups []*ResourceGroup, err error) {
	var result = c.NewLinkedMap()
	for _, v := range p.resouceReg.Entries() {
		id := v.Key.(string)
		resource := v.Value.(*Resource)
		ids := c.SplitTrimOmitEmpty(id, ".")
		if len(ids) > depth {
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

// NewResource creates a resource.
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

// NewResourceAndReg creates and registers a resource, panicking if its ID already exists.
func NewResourceAndReg(registry *ResourceRegistry, name, id string, parent *Resource) *Resource {
	res := NewResource(name, id, parent)
	if err := registry.Add(res); err != nil {
		panic(err)
	}
	return res
}

type permKey int

const (
	required permKey = 0 // Required permissions.
	user     permKey = 1 // Authenticated user.
)

// Perm combines a resource with its allowed operations.
type Perm struct {
	Res *Resource // Resources.
	Op  Operation // Permission operations.
}

// NewPerm creates a permission.
func NewPerm(res *Resource, op Operation) *Perm {
	return &Perm{Res: res, Op: op}
}

// Role defines a named set of permissions.
type Role interface {
	// GetName returns the role name.
	GetName() string
	// GetPerms returns the role's permissions.
	GetPerms() map[string]Operation
}

// Principal identifies a subject with roles and permissions.
type Principal interface {
	// GetID returns the principal ID.
	GetID() int64
	// GetName returns the principal name.
	GetName() string
	// GetRoles returns the principal's roles.
	GetRoles() []Role
}

// ReqPerm adds required permissions to the context.
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

// BindPrincipal stores principal in the context.
func BindPrincipal(ctx context.Context, principal Principal) (context.Context, error) {
	if ctx == nil || principal == nil {
		return ctx, fmt.Errorf("Ctx or principal must not be nil")
	}
	ctx = context.WithValue(ctx, user, principal)
	return ctx, nil
}

// GetPrincipal retrieves the principal from the context.
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

// GetRequiredPerm retrieves the required permissions from the context.
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

// HasPermWithPrinciapl checks whether the principal has the permissions required by the context.
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

// HasPermWithPrincipalAndPerms checks whether the principal has all required permissions.
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
