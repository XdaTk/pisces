package middleware

import (
	"bufio"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"pisces"
	"strings"
)

type (
	// GzipConfig defines the config for Gzip middleware.
	GzipConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// Gzip compression level.
		// Optional. Default value -1.
		Level int `yaml:"level"`
	}

	gzipResponseWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

const (
	gzipScheme = "gzip"
)

var (
	// DefaultGzipConfig is the default Gzip middleware config.
	DefaultGzipConfig = GzipConfig{
		Skipper: DefaultSkipper,
		Level:   -1,
	}
)

// Gzip returns a middleware which compresses HTTP response using gzip compression
// scheme.
func Gzip() pisces.MiddlewareFunc {
	return GzipWithConfig(DefaultGzipConfig)
}

// GzipWithConfig return Gzip middleware with config.
// See: `Gzip()`.
func GzipWithConfig(config GzipConfig) pisces.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultGzipConfig.Skipper
	}
	if config.Level == 0 {
		config.Level = DefaultGzipConfig.Level
	}

	return func(next pisces.HandlerFunc) pisces.HandlerFunc {
		return func(c pisces.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			res := c.Response()
			res.Header().Add(pisces.HeaderVary, pisces.HeaderAcceptEncoding)
			if strings.Contains(c.Request().Header.Get(pisces.HeaderAcceptEncoding), gzipScheme) {
				res.Header().Set(pisces.HeaderContentEncoding, gzipScheme) // Issue #806
				rw := res.Writer
				w, err := gzip.NewWriterLevel(rw, config.Level)
				if err != nil {
					return err
				}
				defer func() {
					if res.Size == 0 {
						if res.Header().Get(pisces.HeaderContentEncoding) == gzipScheme {
							res.Header().Del(pisces.HeaderContentEncoding)
						}
						// We have to reset response to it's pristine state when
						// nothing is written to body or error is returned.
						// See issue #424, #407.
						res.Writer = rw
						w.Reset(ioutil.Discard)
					}
					w.Close()
				}()
				grw := &gzipResponseWriter{Writer: w, ResponseWriter: rw}
				res.Writer = grw
			}
			return next(c)
		}
	}
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if code == http.StatusNoContent { // Issue #489
		w.ResponseWriter.Header().Del(pisces.HeaderContentEncoding)
	}
	w.Header().Del(pisces.HeaderContentLength) // Issue #444
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(pisces.HeaderContentType) == "" {
		w.Header().Set(pisces.HeaderContentType, http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	_ = w.Writer.(*gzip.Writer).Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
