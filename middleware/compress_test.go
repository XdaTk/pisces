package middleware

import (
	"bytes"
	"compress/gzip"
	"github.com/xdatk/pisces"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzip(t *testing.T) {
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Skip if no Accept-Encoding header
	h := Gzip()(func(c pisces.Context) error {
		_, _ = c.Response().Write([]byte("test")) // For Content-Type sniffing
		return nil
	})
	_ = h(c)

	assert := assert.New(t)

	assert.Equal("test", rec.Body.String())

	// Gzip
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(pisces.HeaderAcceptEncoding, gzipScheme)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h(c)
	assert.Equal(gzipScheme, rec.Header().Get(pisces.HeaderContentEncoding))
	assert.Contains(rec.Header().Get(pisces.HeaderContentType), pisces.MIMETextPlain)
	r, err := gzip.NewReader(rec.Body)
	if assert.NoError(err) {
		buf := new(bytes.Buffer)
		defer r.Close()
		_, _ = buf.ReadFrom(r)
		assert.Equal("test", buf.String())
	}

	chunkBuf := make([]byte, 5)

	// Gzip chunked
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(pisces.HeaderAcceptEncoding, gzipScheme)
	rec = httptest.NewRecorder()

	c = e.NewContext(req, rec)
	_ = Gzip()(func(c pisces.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Transfer-Encoding", "chunked")

		// Write and flush the first part of the data
		_, _ = c.Response().Write([]byte("test\n"))
		c.Response().Flush()

		// Read the first part of the data
		assert.True(rec.Flushed)
		assert.Equal(gzipScheme, rec.Header().Get(pisces.HeaderContentEncoding))
		_ = r.Reset(rec.Body)

		_, err = io.ReadFull(r, chunkBuf)
		assert.NoError(err)
		assert.Equal("test\n", string(chunkBuf))

		// Write and flush the second part of the data
		_, _ = c.Response().Write([]byte("test\n"))
		c.Response().Flush()

		_, err = io.ReadFull(r, chunkBuf)
		assert.NoError(err)
		assert.Equal("test\n", string(chunkBuf))

		// Write the final part of the data and return
		_, _ = c.Response().Write([]byte("test"))
		return nil
	})(c)

	buf := new(bytes.Buffer)
	defer r.Close()
	_, _ = buf.ReadFrom(r)
	assert.Equal("test", buf.String())
}

func TestGzipNoContent(t *testing.T) {
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(pisces.HeaderAcceptEncoding, gzipScheme)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := Gzip()(func(c pisces.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	if assert.NoError(t, h(c)) {
		assert.Empty(t, rec.Header().Get(pisces.HeaderContentEncoding))
		assert.Empty(t, rec.Header().Get(pisces.HeaderContentType))
		assert.Equal(t, 0, len(rec.Body.Bytes()))
	}
}

func TestGzipErrorReturned(t *testing.T) {
	e := pisces.New()
	e.Use(Gzip())
	e.GET("/", func(c pisces.Context) error {
		return pisces.ErrNotFound
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(pisces.HeaderAcceptEncoding, gzipScheme)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get(pisces.HeaderContentEncoding))
}

// Issue #806
func TestGzipWithStatic(t *testing.T) {
	e := pisces.New()
	e.Use(Gzip())
	e.Static("/test", "../_fixture/images")
	req := httptest.NewRequest(http.MethodGet, "/test/walle.png", nil)
	req.Header.Set(pisces.HeaderAcceptEncoding, gzipScheme)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Data is written out in chunks when Content-Length == "", so only
	// validate the content length if it's not set.
	if cl := rec.Header().Get("Content-Length"); cl != "" {
		assert.Equal(t, cl, rec.Body.Len())
	}
	r, err := gzip.NewReader(rec.Body)
	if assert.NoError(t, err) {
		defer r.Close()
		want, err := ioutil.ReadFile("../_fixture/images/walle.png")
		if assert.NoError(t, err) {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(r)
			assert.Equal(t, want, buf.Bytes())
		}
	}
}
