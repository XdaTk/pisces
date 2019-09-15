package middleware

import (
	"github.com/xdatk/pisces"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	e := pisces.New()

	// Wildcard origin
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := CORS()(pisces.NotFoundHandler)
	_ = h(c)
	assert.Equal(t, "*", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))

	// Allow origins
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = CORSWithConfig(CORSConfig{
		AllowOrigins: []string{"localhost"},
	})(pisces.NotFoundHandler)
	req.Header.Set(pisces.HeaderOrigin, "localhost")
	_ = h(c)
	assert.Equal(t, "localhost", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))

	// Preflight request
	req = httptest.NewRequest(http.MethodOptions, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	req.Header.Set(pisces.HeaderOrigin, "localhost")
	req.Header.Set(pisces.HeaderContentType, pisces.MIMEApplicationJSON)
	cors := CORSWithConfig(CORSConfig{
		AllowOrigins:     []string{"localhost"},
		AllowCredentials: true,
		MaxAge:           3600,
	})
	h = cors(pisces.NotFoundHandler)
	_ = h(c)
	assert.Equal(t, "localhost", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))
	assert.NotEmpty(t, rec.Header().Get(pisces.HeaderAccessControlAllowMethods))
	assert.Equal(t, "true", rec.Header().Get(pisces.HeaderAccessControlAllowCredentials))
	assert.Equal(t, "3600", rec.Header().Get(pisces.HeaderAccessControlMaxAge))

	// Preflight request with `AllowOrigins` *
	req = httptest.NewRequest(http.MethodOptions, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	req.Header.Set(pisces.HeaderOrigin, "localhost")
	req.Header.Set(pisces.HeaderContentType, pisces.MIMEApplicationJSON)
	cors = CORSWithConfig(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           3600,
	})
	h = cors(pisces.NotFoundHandler)
	_ = h(c)
	assert.Equal(t, "localhost", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))
	assert.NotEmpty(t, rec.Header().Get(pisces.HeaderAccessControlAllowMethods))
	assert.Equal(t, "true", rec.Header().Get(pisces.HeaderAccessControlAllowCredentials))
	assert.Equal(t, "3600", rec.Header().Get(pisces.HeaderAccessControlMaxAge))

	// Preflight request with `AllowOrigins` which allow all subdomains with *
	req = httptest.NewRequest(http.MethodOptions, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	req.Header.Set(pisces.HeaderOrigin, "http://aaa.example.com")
	cors = CORSWithConfig(CORSConfig{
		AllowOrigins: []string{"http://*.example.com"},
	})
	h = cors(pisces.NotFoundHandler)
	_ = h(c)
	assert.Equal(t, "http://aaa.example.com", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))

	req.Header.Set(pisces.HeaderOrigin, "http://bbb.example.com")
	_ = h(c)
	assert.Equal(t, "http://bbb.example.com", rec.Header().Get(pisces.HeaderAccessControlAllowOrigin))
}
