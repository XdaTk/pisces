package pisces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	p := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := p.NewContext(req, rec)
	res := &Response{pisces: p, Writer: rec}

	// Before
	res.Before(func() {
		c.Response().Header().Set(HeaderServer, "pisces")
	})
	_, _ = res.Write([]byte("test"))
	assert.Equal(t, "pisces", rec.Header().Get(HeaderServer))
}

func TestResponse_Write_FallsBackToDefaultStatus(t *testing.T) {
	p := New()
	rec := httptest.NewRecorder()
	res := &Response{pisces: p, Writer: rec}

	_, _ = res.Write([]byte("test"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponse_Write_UsesSetResponseCode(t *testing.T) {
	e := New()
	rec := httptest.NewRecorder()
	res := &Response{pisces: e, Writer: rec}

	res.Status = http.StatusBadRequest
	_, _ = res.Write([]byte("test"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
