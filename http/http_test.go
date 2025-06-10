package http

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockController struct {
	BaseController
}

func (p *MockController) Index(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Write([]byte("Error:" + err.Error()))
		return
	}
	ret := fmt.Sprintf("method:%s, param id:%s, path id:%s", r.Method, r.FormValue("id"), r.PathValue("id"))
	w.Write([]byte(ret))
}

func TestHttpServer(t *testing.T) {
	controller := &MockController{
		BaseController: BaseController{
			Name: "Mock",
			Path: "/",
			PatternMethods: map[string]string{
				"/":                       "Index",
				"localhost/index/{id...}": "Index",
				"/index/{id}":             "Index",
			},
		},
	}

	httpConfig := NewConfig("127.0.0.1:8888")
	err := httpConfig.RegController(controller)
	assert.NoError(t, err)

	httpSvc := &Service{

		Conf: httpConfig,
	}
	ok := httpSvc.Init()
	assert.True(t, ok)

	ok = httpSvc.Start()
	assert.True(t, ok)
	defer httpSvc.Stop()

	client := &http.Client{}
	ret, err := GetURL(client, "http://localhost:8888", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:, path id:", ret)

	ret, err = GetURL(client, "http://localhost:8888/index/id1?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:id1", ret)

	ret, err = GetURL(client, "http://127.0.0.1:8888/index/id3?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:id3", ret)

	ret, err = GetURL(client, "http://127.0.0.1:8888/index/?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:", ret)
}
