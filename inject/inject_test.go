package inject

import (
	_ "fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type accountService interface {
	Name() string
}

type userRegService struct {
	N        int
	LdapImpl accountService `inject:"ldap"`
	DbImpl   accountService `inject:"db"`
	Id       string
}

type ldapAccount struct {
	n string
}

func (self *ldapAccount) Name() string {
	return self.n + "@ldap"
}

type dbAccount struct {
	n string
}

func (self *dbAccount) Name() string {
	return self.n + "@db"
}

func TestInject(t *testing.T) {
	ldapImplA := ldapAccount{n: "a"}
	dbImplA := dbAccount{n: "b"}

	user := "user_name"

	mod := NewModule()
	mod.BindWithName("ldap", &ldapImplA)
	mod.BindWithName("db", &dbImplA)
	mod.Bind(user)
	mod.Bind(&user)

	injector := NewInjector([]*Module{mod})

	regService := &userRegService{}
	injector.RequireInject(regService)
	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "b@db", regService.DbImpl.Name())

	injector.RequireInjectWithOverrideTags(regService, map[string]string{"DbImpl": "ldap", "LdapImpl": "db"})
	assert.Equal(t, "b@db", regService.LdapImpl.Name())
	assert.Equal(t, "a@ldap", regService.DbImpl.Name())

	injector.RequireInject(regService)
	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "b@db", regService.DbImpl.Name())

	//根据名称查找
	ldapImplGet := injector.GetInstanceByPrototype("ldap", struct{ s accountService }{}).(accountService)
	assert.NotNil(t, ldapImplGet)
	assert.Equal(t, &ldapImplA, ldapImplGet)
	assert.Equal(t, "a@ldap", ldapImplGet.Name())

	ldapImplGet, ok := injector.GetInstanceByPrototype("", struct{ s accountService }{}).(accountService)
	assert.False(t, ok)
	assert.Nil(t, ldapImplGet)
}

func TestInjectInModule(t *testing.T) {
	ldapImplA := ldapAccount{n: "a"}
	dbImplA := dbAccount{n: "b"}
	regService := &userRegService{}

	user := "user_name"
	mod := NewModule()
	mod.BindWithName("ldap", &ldapImplA)
	mod.BindWithName("db", &dbImplA)
	mod.Bind(user)
	mod.Bind(regService)

	_ = NewInjector([]*Module{mod})

	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "b@db", regService.DbImpl.Name())
}

func TestInjectInModuleWithTag(t *testing.T) {
	ldapImplA := ldapAccount{n: "a"}
	dbImplA := dbAccount{n: "b"}
	regService := &userRegService{}

	user := "user_name"
	mod := NewModule()
	mod.BindWithName("ldap", &ldapImplA)
	mod.BindWithName("db", &dbImplA)
	mod.Bind(user)
	mod.BindWithNameOverrideTags("", regService, map[string]string{"DbImpl": "ldap"})

	_ = NewInjector([]*Module{mod})

	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "a@ldap", regService.DbImpl.Name())
}

func TestInjectInModuleWithProviderFunc(t *testing.T) {
	ldapImplA := ldapAccount{n: "a"}
	dbImplA := dbAccount{n: "b"}
	regService := &userRegService{}

	user := "user_name"
	mod := NewModule()
	mod.BindWithProviderFunc("ldap", func() interface{} {
		return &ldapImplA
	})
	mod.BindWithProviderFunc("db", func() interface{} {
		return &dbImplA
	})
	mod.Bind(user)
	mod.BindWithNameOverrideTags("", regService, map[string]string{"DbImpl": "ldap"})

	_ = NewInjector([]*Module{mod})

	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "a@ldap", regService.DbImpl.Name())
}

type ldapProvider struct{}

func (self ldapProvider) GetInstance() interface{} {
	return &ldapAccount{n: "a"}
}

type dbProvider struct{}

func (self dbProvider) GetInstance() interface{} {
	return &dbAccount{n: "b"}
}

func TestInjectInModuleWithProvider(t *testing.T) {
	regService := &userRegService{}

	user := "user_name"
	mod := NewModule()
	mod.BindWithProvider("ldap", ldapProvider{})
	mod.BindWithProvider("db", dbProvider{})
	mod.Bind(user)
	mod.BindWithNameOverrideTags("", regService, map[string]string{"DbImpl": "ldap"})

	_ = NewInjector([]*Module{mod})

	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "a@ldap", regService.DbImpl.Name())
}

func TestGetInstancesByPrototype(t *testing.T) {
	ldapImplA := ldapAccount{n: "a"}
	dbImplA := dbAccount{n: "b"}
	regService := &userRegService{}

	user := "user_name"
	mod := NewModule()
	mod.BindWithName("ldap", &ldapImplA)
	mod.BindWithName("db", &dbImplA)
	mod.Bind(user)
	mod.Bind(regService)

	injector := NewInjector([]*Module{mod})

	assert.Equal(t, "a@ldap", regService.LdapImpl.Name())
	assert.Equal(t, "b@db", regService.DbImpl.Name())

	allAccountServices := injector.GetInstancesByPrototype(struct{ a accountService }{})
	assert.Equal(t, 2, len(allAccountServices))

	for _, v := range allAccountServices {
		_ = v.(accountService)
	}
}
