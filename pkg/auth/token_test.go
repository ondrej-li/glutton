package auth

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTokenProvider_Generate1(t *testing.T) {
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test", time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	log.Printf("token: [%s]", token)
}

func TestDefaultTokenProvider_Validate1(t *testing.T) {
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test", time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	valid, err := generator.ValidateToken(token, "test", time.Now())
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestDefaultTokenProvider_Validate2(t *testing.T) {
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test1", time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	valid, err := generator.ValidateToken(token, "test2", time.Now())
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestDefaultTokenProvider_Validate3(t *testing.T) {
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test1", time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	valid, err := generator.ValidateToken(token, "test1", time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestDefaultTokenProvider_Validate4(t *testing.T) {
	// a token must stay valid for the whole maximum duration (regression for the inverted check)
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test", time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	valid, err := generator.ValidateToken(token, "test", time.Now().Add(30*time.Second))
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestDefaultTokenProvider_Validate5(t *testing.T) {
	// a token dated in the future must be rejected
	generator := NewDefaultTokenProvider(time.Minute, []byte(""), true)
	token, err := generator.GenerateToken("test", time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	valid, err := generator.ValidateToken(token, "test", time.Now())
	assert.NoError(t, err)
	assert.False(t, valid)
}
