package middleware

import (
	"bufio"
	"bytes"
	"github.com/xdatk/pisces"
	"io"
	"io/ioutil"
	"net"
	"net/http"
)

type (
	// BodyDumpConfig defines the config for BodyDump middleware.
	BodyDumpConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// Handler receives request and response payload.
		// Required.
		Handler BodyDumpHandler
	}

	// BodyDumpHandler receives the request and response payload.
	BodyDumpHandler func(pisces.Context, []byte, []byte)

	bodyDumpResponseWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultBodyDumpConfig = BodyDumpConfig{
		Skipper: DefaultSkipper,
	}
)

// BodyDump returns a BodyDump middleware.
//
// BodyLimit middleware captures the request and response payload and calls the
// registered handler.
func BodyDump(handler BodyDumpHandler) pisces.MiddlewareFunc {
	c := DefaultBodyDumpConfig
	c.Handler = handler
	return BodyDumpWithConfig(c)
}

// BodyDumpWithConfig returns a BodyDump middleware with config.
// See: `BodyDump()`.
func BodyDumpWithConfig(config BodyDumpConfig) pisces.MiddlewareFunc {
	// Defaults
	if config.Handler == nil {
		panic("pisces: body-dump middleware requires a handler function")
	}
	if config.Skipper == nil {
		config.Skipper = DefaultBodyDumpConfig.Skipper
	}

	return func(next pisces.HandlerFunc) pisces.HandlerFunc {
		return func(c pisces.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			// Request
			reqBody := []byte{}
			if c.Request().Body != nil { // Read
				reqBody, _ = ioutil.ReadAll(c.Request().Body)
			}
			c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset

			// Response
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			if err = next(c); err != nil {
				c.Error(err)
			}

			// Callback
			config.Handler(c, reqBody, resBody.Bytes())

			return
		}
	}
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
