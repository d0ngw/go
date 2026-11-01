package http

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	httpConfig := NewConfig("127.0.0.1:0")
	err := httpConfig.RegController(controller)
	assert.NoError(t, err)

	httpSvc := &Service{

		Conf: httpConfig,
	}
	require.NoError(t, httpSvc.Init())

	require.True(t, httpSvc.Start())
	defer httpSvc.Stop()

	client := &http.Client{}
	t.Cleanup(client.CloseIdleConnections)
	addr := httpSvc.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	ret, err := GetURL(client, "http://localhost:"+port, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:, path id:", ret)

	ret, err = GetURL(client, "http://localhost:"+port+"/index/id1?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:id1", ret)

	ret, err = GetURL(client, "http://"+addr+"/index/id3?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:id3", ret)

	ret, err = GetURL(client, "http://"+addr+"/index/?id=id2", nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "method:GET, param id:id2, path id:", ret)
}
