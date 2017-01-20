package http

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type DemoController struct {
	BaseController
}

func (self *DemoController) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Index:", self.Path, self.Name)
}

func (self *DemoController) Second(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Second:", self.Path, self.Name)
}

func (self *DemoController) GetHandlers() (map[string]http.HandlerFunc, error) {
	return ReflectHandlers(self)
}

func TestReflectHandlers(t *testing.T) {
	testReflectHandlers(t, "demo1")
	testReflectHandlers(t, "demo2")
}

func testReflectHandlers(t *testing.T, name string) {
	controller := &DemoController{
		BaseController: BaseController{
			Name: name,
			Path: "/" + name,
		},
	}

	mapping, err := ReflectHandlers(controller)
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 2, len(mapping))

	mapping["index"](nil, nil)
	mapping["second"](nil, nil)

	mapping, err = controller.GetHandlers()
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 2, len(mapping))
	mapping["second"](nil, nil)
	mapping["index"](nil, nil)
}

func TestToUnderlineName(t *testing.T) {
	assert.EqualValues(t, "index", ToUnderlineName("index"))
	assert.EqualValues(t, "index", ToUnderlineName("INDEX"))
	assert.EqualValues(t, "index", ToUnderlineName("Index"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDex"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDEX"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDEx"))
	assert.EqualValues(t, "in_de_x", ToUnderlineName("InDeX"))
	assert.EqualValues(t, "in语言de_x", ToUnderlineName("In语言DeX"))
	assert.EqualValues(t, "", ToUnderlineName(""))
}
