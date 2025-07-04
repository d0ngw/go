package http

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DemoController struct {
	BaseController
}

func (p *DemoController) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Index:", p.Path, p.Name)
}

func (p *DemoController) Second(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Second:", p.Path, p.Name)
}

func TestReflectHandlers(t *testing.T) {
	testReflectHandlers(t, "demo1")
	testReflectHandlers(t, "demo2")
	testReflectPatternsHandlers(t, "demo1pattern")
	testReflectPatternsHandlers(t, "demo2pattern")
}

type LogMiddleware struct {
	Order int
}

func NewLogMiddleware(order int) *LogMiddleware {
	return &LogMiddleware{
		Order: order,
	}
}

func (p *LogMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Begin process,order", p.Order)
		next(w, r)
		fmt.Println("Finish process,order", p.Order)
	}
}

func testReflectHandlers(t *testing.T, name string) {
	controller := &DemoController{
		BaseController: BaseController{
			Name: name,
			Path: "/" + name,
			HandlerMiddlewares: map[string][]Middleware{
				"Index": {
					&LogMiddleware{Order: 0},
					&LogMiddleware{Order: 1},
					&LogMiddleware{Order: 2},
				},
			},
		},
	}

	mapping, err := reflectHandlers(controller)
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 2, len(mapping))

	mapping[controller.Path+"/index"].handlerFunc(nil, nil)
	mapping[controller.Path+"/second"].handlerFunc(nil, nil)

	mapping, err = reflectHandlers(controller)
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 2, len(mapping))
	mapping[controller.Path+"/second"].handlerFunc(nil, nil)
	mapping[controller.Path+"/index"].handlerFunc(nil, nil)
}

func testReflectPatternsHandlers(t *testing.T, name string) {
	controller := &DemoController{
		BaseController: BaseController{
			Name: name,
			Path: "/",
			HandlerMiddlewares: map[string][]Middleware{
				"Index": {
					&LogMiddleware{Order: 0},
					&LogMiddleware{Order: 1},
					&LogMiddleware{Order: 2},
				},
			},
			PatternMethods: map[string]string{
				"/":                "Index",
				"GET /":            "Index",
				"GET localhost/":   "Index",
				"/index/{id}":      "Index",
				"/sec/{id}/{name}": "Second",
			},
		},
	}

	mapping, err := reflectHandlers(controller)
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 5, len(mapping))

	mapping[controller.Path].handlerFunc(nil, nil)
	mapping["GET "+controller.Path].handlerFunc(nil, nil)
	mapping["GET localhost"+controller.Path].handlerFunc(nil, nil)
	mapping[controller.Path+"index/{id}"].handlerFunc(nil, nil)
	mapping[controller.Path+"sec/{id}/{name}"].handlerFunc(nil, nil)

	controller.Path = "/mock"
	mapping, err = reflectHandlers(controller)
	assert.Nil(t, err, "err")
	assert.EqualValues(t, 5, len(mapping))

	mapping[controller.Path].handlerFunc(nil, nil)
	mapping["GET "+controller.Path].handlerFunc(nil, nil)
	mapping["GET localhost"+controller.Path].handlerFunc(nil, nil)
	mapping[controller.Path+"/index/{id}"].handlerFunc(nil, nil)
	mapping[controller.Path+"/sec/{id}/{name}"].handlerFunc(nil, nil)
}

func TestToUnderlineName(t *testing.T) {
	assert.EqualValues(t, "index", ToUnderlineName("index"))
	assert.EqualValues(t, "index", ToUnderlineName("INDEX"))
	assert.EqualValues(t, "index", ToUnderlineName("Index"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDex"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDEX"))
	assert.EqualValues(t, "in_dex", ToUnderlineName("InDEx"))
	assert.EqualValues(t, "in_de_x", ToUnderlineName("InDeX"))
	assert.EqualValues(t, "in语言_de_x", ToUnderlineName("In语言DeX"))
	assert.EqualValues(t, "", ToUnderlineName(""))
	assert.EqualValues(t, "h5_ware_path", ToUnderlineName("H5WarePath"))
}
