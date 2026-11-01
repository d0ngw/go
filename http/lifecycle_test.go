package http

import (
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/inject"
	"github.com/stretchr/testify/require"
)

func TestHTTPServiceInjectorLifecycle(t *testing.T) {
	conf := NewConfig("127.0.0.1:0")
	require.NoError(t, conf.RegHandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ready")) }))
	svc := &Service{Conf: conf}
	mod := inject.NewModule()
	mod.Bind(svc)
	inj := inject.NewInjector([]*inject.Module{mod})
	require.NoError(t, inj.Initialize())
	require.Equal(t, c.INITED, svc.State())
	require.NoError(t, inj.Start())
	t.Cleanup(func() { svc.Stop() })
	require.Equal(t, c.RUNNING, svc.State())
	client := &http.Client{Timeout: 2 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	resp, err := client.Get("http://" + svc.Addr().String())
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, "ready", string(body))
	// Shutdown must close the idle keep-alive connection without waiting for the client.
	require.NoError(t, inj.Stop())
	require.Equal(t, c.TERMINATED, svc.State())
	require.Nil(t, svc.Addr())
}

func TestHTTPServiceInvalidConfiguration(t *testing.T) {
	require.Error(t, (&Service{}).Init())
	conf := NewConfig("127.0.0.1:0")
	require.NoError(t, conf.RegHandleFunc("/", nil))
	require.Error(t, (&Service{Conf: conf}).Init())
	conf = NewConfig("127.0.0.1:0")
	require.NoError(t, conf.RegHandleFunc("/{invalid", func(http.ResponseWriter, *http.Request) {}))
	require.Error(t, (&Service{Conf: conf}).Init())
	var svc Service
	require.False(t, svc.Start())
	require.True(t, svc.Stop())
	require.True(t, svc.Stop())
	svc.Conf = NewConfig("127.0.0.1:0")
	require.NoError(t, svc.Init())
	require.True(t, svc.Stop())
	require.True(t, svc.Stop())
}

func TestHTTPServiceClosesIdleConnections(t *testing.T) {
	conf := NewConfig("127.0.0.1:0")
	require.NoError(t, conf.RegHandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	svc := &Service{Conf: conf}
	require.NoError(t, svc.Init())
	closed := make(chan struct{}, 1)
	svc.server.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			closed <- struct{}{}
		}
	}
	require.True(t, svc.Start())
	t.Cleanup(func() { svc.Stop() })
	client := &http.Client{Timeout: 2 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	resp, err := client.Get("http://" + svc.Addr().String())
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.True(t, svc.Stop())
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("idle connection was not closed")
	}
}

func TestHTTPServiceDrainsRequests(t *testing.T) {
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	conf := NewConfig("127.0.0.1:0")
	conf.ShutdownTimeout = 2 * time.Second
	require.NoError(t, conf.RegHandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.Write([]byte("done"))
	}))
	svc := &Service{Conf: conf}
	require.NoError(t, svc.Init())
	require.True(t, svc.Start())
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }); svc.Stop() })
	go func() {
		defer close(finished)
		resp, err := http.Get("http://" + svc.Addr().String())
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
	stopped := make(chan bool, 1)
	go func() { stopped <- svc.Stop() }()
	select {
	case <-stopped:
		t.Fatal("shutdown returned before the request completed")
	case <-time.After(20 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case ok := <-stopped:
		require.True(t, ok)
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not complete")
	}
	<-finished
}

func TestHTTPServiceShutdownDeadlineAndConcurrentStop(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	conf := NewConfig("127.0.0.1:0")
	conf.ShutdownTimeout = 30 * time.Millisecond
	require.NoError(t, conf.RegHandleFunc("/", func(w http.ResponseWriter, r *http.Request) { close(entered); <-release }))
	svc := &Service{Conf: conf}
	require.NoError(t, svc.Init())
	require.True(t, svc.Start())
	t.Cleanup(func() { close(release); svc.Stop() })
	go func() {
		resp, err := http.Get("http://" + svc.Addr().String())
		if err == nil {
			resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
	results := make(chan bool, 8)
	for i := 0; i < cap(results); i++ {
		go func() { results <- svc.Stop() }()
	}
	for i := 0; i < cap(results); i++ {
		select {
		case ok := <-results:
			require.False(t, ok)
		case <-time.After(2 * time.Second):
			t.Fatal("shutdown exceeded its deadline")
		}
	}
	require.False(t, svc.Stop())
}
