package middleware

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"pisces"
	"testing"
)

func TestRecover(t *testing.T) {
	e := pisces.New()
	buf := new(bytes.Buffer)
	e.Logger.SetOutput(buf)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := Recover()(pisces.HandlerFunc(func(c pisces.Context) error {
		panic("test")
	}))
	h(c)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, buf.String(), "PANIC RECOVER")
}
