package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/lyft/flyteadmin/pkg/auth/config"
	"github.com/lyft/flyteadmin/pkg/auth/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

// This function can also be called locally to generate new keys
func TestSecureCookieLifecycle(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(64)
	assert.True(t, base64.RawStdEncoding.EncodeToString(hashKey) != "")

	blockKey := securecookie.GenerateRandomKey(32)
	assert.True(t, base64.RawStdEncoding.EncodeToString(blockKey) != "")
	fmt.Printf("Hash key: |%s| Block key: |%s|\n",
		base64.RawStdEncoding.EncodeToString(hashKey), base64.RawStdEncoding.EncodeToString(blockKey))

	cookie, err := NewSecureCookie("choc", "chip", hashKey, blockKey)
	assert.NoError(t, err)

	value, err := ReadSecureCookie(context.Background(), cookie, hashKey, blockKey)
	assert.NoError(t, err)
	assert.Equal(t, "chip", value)
}

func TestNewCsrfToken(t *testing.T) {
	csrf := NewCsrfToken(5)
	assert.Equal(t, "5qz3p9w8qo", csrf)
}

func TestNewCsrfCookie(t *testing.T) {
	cookie := NewCsrfCookie()
	assert.Equal(t, "flyte_csrf_state", cookie.Name)
	assert.True(t, cookie.HttpOnly)
}

func TestHashCsrfState(t *testing.T) {
	h := HashCsrfState("hello world")
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", h)
}

func TestVerifyCsrfCookie(t *testing.T) {
	t.Run("test no state", func(t *testing.T) {
		var buf bytes.Buffer
		request, err := http.NewRequest(http.MethodPost, "/test", &buf)
		assert.NoError(t, err)
		valid := VerifyCsrfCookie(context.Background(), request)
		assert.False(t, valid)
	})

	t.Run("test incorrect token", func(t *testing.T) {
		var buf bytes.Buffer
		request, err := http.NewRequest(http.MethodPost, "/test", &buf)
		assert.NoError(t, err)
		v := url.Values{
			"state": []string{"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"},
		}
		cookie := NewCsrfCookie()
		cookie.Value = "helloworld"
		request.Form = v
		request.AddCookie(&cookie)
		valid := VerifyCsrfCookie(context.Background(), request)
		assert.False(t, valid)
	})

	t.Run("test correct token", func(t *testing.T) {
		var buf bytes.Buffer
		request, err := http.NewRequest(http.MethodPost, "/test", &buf)
		assert.NoError(t, err)
		v := url.Values{
			"state": []string{"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"},
		}
		cookie := NewCsrfCookie()
		cookie.Value = "hello world"
		request.Form = v
		request.AddCookie(&cookie)
		valid := VerifyCsrfCookie(context.Background(), request)
		assert.True(t, valid)
	})
}

func TestNewRedirectCookie(t *testing.T) {
	ctx := context.Background()
	cookie := NewRedirectCookie(ctx, "/console")
	assert.NotNil(t, cookie)
	assert.Equal(t, "/console", cookie.Value)
}

func TestGetAuthFlowEndRedirect(t *testing.T) {
	t.Run("in request", func(t *testing.T) {
		ctx := context.Background()
		request, err := http.NewRequest(http.MethodGet, "/test", nil)
		assert.NoError(t, err)
		cookie := NewRedirectCookie(ctx, "/console")
		assert.NotNil(t, cookie)
		request.AddCookie(cookie)
		mockAuthCtx := &mocks.AuthenticationContext{}
		redirect := getAuthFlowEndRedirect(ctx, mockAuthCtx, request)
		assert.Equal(t, "/console", redirect)
	})

	t.Run("not in request", func(t *testing.T) {
		ctx := context.Background()
		request, err := http.NewRequest(http.MethodGet, "/test", nil)
		assert.NoError(t, err)
		mockAuthCtx := &mocks.AuthenticationContext{}
		mockAuthCtx.On("Options").Return(config.OAuthOptions{
			RedirectUrl: "/api/v1/projects",
		})
		redirect := getAuthFlowEndRedirect(ctx, mockAuthCtx, request)
		assert.Equal(t, "/api/v1/projects", redirect)
	})
}