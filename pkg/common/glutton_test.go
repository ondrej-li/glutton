package common

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServeReportsErrorWhenPortInUse(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	err = serve(context.Background(), gin.New(), listener.Addr().String())
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
		done <- serve(ctx, gin.New(), address)
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
