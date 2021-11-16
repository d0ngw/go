package inject

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type config struct {
	Name string `yaml:"name"`

	needModule bool
}

func (p *config) Parse() error {
	return nil
}

func (p *config) ConfModule() (module *Module, err error) {
	if !p.needModule {
		return
	}
	module = NewModule()
	module.BindWithName("hello", 2021)
	return
}

type userService struct {
	Injector *Injector `inject:"_"`
	Hello    int       `inject:"hello,optional"`
}

func (p *userService) Init() error {
	if p.Injector == nil {
		return errors.New("invalid Injector")
	}
	return nil
}

func TestSetupInjector(t *testing.T) {
	conf := &config{}
	module := NewModule()
	svc := &userService{}
	module.Bind(svc)

	err := os.Chdir("testdata")
	injector, err := SetupInjector(conf, "", "dev", module)
	assert.NoError(t, err)
	assert.NotNil(t, injector)
	assert.NotNil(t, svc.Injector)
	assert.EqualValues(t, 0, svc.Hello)

	conf.needModule = true
	svc = &userService{}
	module = NewModule()
	module.Bind(svc)
	injector, err = SetupInjector(conf, "", "dev", module)
	assert.NoError(t, err)
	assert.NotNil(t, injector)
	assert.NotNil(t, svc.Injector)
	assert.EqualValues(t, 2021, svc.Hello)
}
