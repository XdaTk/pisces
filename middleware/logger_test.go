package middleware

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"pisces"
	"testing"
)

func TestLogger(t *testing.T) {
	// Note: Just for the test coverage, not a real test.
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := Logger()(func(c pisces.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// Status 2xx
	_ = h(c)

	// Status 3xx
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = Logger()(func(c pisces.Context) error {
		return c.String(http.StatusTemporaryRedirect, "test")
	})
	_ = h(c)

	// Status 4xx
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = Logger()(func(c pisces.Context) error {
		return c.String(http.StatusNotFound, "test")
	})
	_ = h(c)

	// Status 5xx with empty path
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = Logger()(func(c pisces.Context) error {
		return errors.New("error")
	})
	_ = h(c)
}

func TestLoggerIPAddress(t *testing.T) {
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	buf := new(bytes.Buffer)
	e.Logger.SetOutput(buf)
	ip := "127.0.0.1"
	h := Logger()(func(c pisces.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// With X-Real-IP
	req.Header.Add(pisces.HeaderXRealIP, ip)
	_ = h(c)
	assert.Contains(t, ip, buf.String())

	// With X-Forwarded-For
	buf.Reset()
	req.Header.Del(pisces.HeaderXRealIP)
	req.Header.Add(pisces.HeaderXForwardedFor, ip)
	_ = h(c)
	assert.Contains(t, ip, buf.String())

	buf.Reset()
	_ = h(c)
	assert.Contains(t, ip, buf.String())
}
