package common

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServeReportsErrorWhenPortInUse(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	err = serve(context.Background(), gin.New(), listener.Addr().String(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error running server")
}

func TestServeReturnsOnContextCancellation(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	address := listener.Addr().String()
	// release the port so that serve is able to bind it
	assert.NoError(t, listener.Close())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, gin.New(), address, nil)
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
}

func TestCORSHeadersOnRequest(t *testing.T) {
	env := CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{URI: "save", OutputFolder: t.TempDir()},
		},
	}, nil)
	req, _ := http.NewRequest("POST", "http://localhost/v1/glutton/save", strings.NewReader("payload"))
	req.Header.Set("Origin", "http://example.com")
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestCORSPreflight(t *testing.T) {
	env := CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{URI: "save", OutputFolder: t.TempDir()},
		},
	}, nil)
	req, _ := http.NewRequest("OPTIONS", "http://localhost/v1/glutton/save", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "token")
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusNoContent, w.Code) &&
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin")) &&
			assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Token")
	})
}
