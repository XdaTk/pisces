package middleware

import (
	"github.com/xdatk/pisces"
	"github.com/xdatk/pisces/middleware/random"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRF(t *testing.T) {
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	csrf := CSRFWithConfig(CSRFConfig{
		TokenLength: 16,
	})
	h := csrf(func(c pisces.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// Generate CSRF token
	_ = h(c)
	assert.Contains(t, rec.Header().Get(pisces.HeaderSetCookie), "_csrf")

	// Without CSRF cookie
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	assert.Error(t, h(c))

	// Empty/invalid CSRF token
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	req.Header.Set(pisces.HeaderXCSRFToken, "")
	assert.Error(t, h(c))

	// Valid CSRF token
	token := random.String(16)
	req.Header.Set(pisces.HeaderCookie, "_csrf="+token)
	req.Header.Set(pisces.HeaderXCSRFToken, token)
	if assert.NoError(t, h(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestCSRFTokenFromForm(t *testing.T) {
	f := make(url.Values)
	f.Set("csrf", "token")
	e := pisces.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(f.Encode()))
	req.Header.Add(pisces.HeaderContentType, pisces.MIMEApplicationForm)
	c := e.NewContext(req, nil)
	token, err := csrfTokenFromForm("csrf")(c)
	if assert.NoError(t, err) {
		assert.Equal(t, "token", token)
	}
	_, err = csrfTokenFromForm("invalid")(c)
	assert.Error(t, err)
}

func TestCSRFTokenFromQuery(t *testing.T) {
	q := make(url.Values)
	q.Set("csrf", "token")
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(pisces.HeaderContentType, pisces.MIMEApplicationForm)
	req.URL.RawQuery = q.Encode()
	c := e.NewContext(req, nil)
	token, err := csrfTokenFromQuery("csrf")(c)
	if assert.NoError(t, err) {
		assert.Equal(t, "token", token)
	}
	_, err = csrfTokenFromQuery("invalid")(c)
	assert.Error(t, err)
	csrfTokenFromQuery("csrf")
}
