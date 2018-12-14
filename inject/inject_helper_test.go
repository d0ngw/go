package inject

import (
	"errors"
	"os"
	"testing"

	c "github.com/d0ngw/go/common"
	"github.com/stretchr/testify/assert"
)

type config struct {
	Name string `yaml:"name"`
}

func (p *config) Parse() error {
	return nil
}

type userService struct {
	Injector *Injector `inject:"_"`
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

	oldEnv := os.Getenv(c.EnvWorkfDir)
	defer func() {
		os.Setenv(c.EnvWorkfDir, oldEnv)
	}()
	os.Setenv(c.EnvWorkfDir, "testdata")

	injector, err := SetupInjector(conf, "", "dev", module)
	assert.NoError(t, err)
	assert.NotNil(t, injector)
	assert.NotNil(t, svc.Injector)
}
