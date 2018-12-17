package pisces

import (
	"bytes"
	stdContext "context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"time"
)

const (
	charsetUTF8 = "charset=UTF-8"
	PROPFIND    = "PROPFIND"
)

const (
	MIMEApplicationJSON            = "application/json"
	MIMEApplicationJSONCharsetUTF8 = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationForm            = "application/x-www-form-urlencoded"
	MIMETextHTML                   = "text/html"
	MIMETextHTMLCharsetUTF8        = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                  = "text/plain"
	MIMETextPlainCharsetUTF8       = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm              = "multipart/form-data"
	MIMEOctetStream                = "application/octet-stream"
)

// Headers
const (
	HeaderAccept              = "Accept"
	HeaderAcceptEncoding      = "Accept-Encoding"
	HeaderAllow               = "Allow"
	HeaderAuthorization       = "Authorization"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderContentEncoding     = "Content-Encoding"
	HeaderContentLength       = "Content-Length"
	HeaderContentType         = "Content-Type"
	HeaderCookie              = "Cookie"
	HeaderSetCookie           = "Set-Cookie"
	HeaderIfModifiedSince     = "If-Modified-Since"
	HeaderLastModified        = "Last-Modified"
	HeaderLocation            = "Location"
	HeaderUpgrade             = "Upgrade"
	HeaderVary                = "Vary"
	HeaderWWWAuthenticate     = "WWW-Authenticate"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXUrlScheme          = "X-Url-Scheme"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-IP"
	HeaderXRequestID          = "X-Request-ID"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderServer              = "Server"
	HeaderOrigin              = "Origin"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

type (
	// Pisces is the top-level framework instance.
	Pisces struct {
		premiddleware    []MiddlewareFunc
		middleware       []MiddlewareFunc
		maxParam         *int
		router           *Router
		notFoundHandler  HandlerFunc
		pool             sync.Pool
		Server           *http.Server
		Listener         net.Listener
		TLSServer        *http.Server
		TLSListener      net.Listener
		HTTPErrorHandler HTTPErrorHandler
		StdLogger        *log.Logger
		Logger           *logrus.Logger
	}

	// Map defines a generic map of type `map[string]interface{}`.
	Map map[string]interface{}

	// HandlerFunc defines a function to serve HTTP requests.
	HandlerFunc func(Context) error

	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(HandlerFunc) HandlerFunc

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Name   string `json:"name"`
	}

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code     int
		Message  interface{}
		Internal error // Stores the error returned by an external dependency
	}

	// i is the interface for Pisces and Group.
	i interface {
		GET(string, HandlerFunc, ...MiddlewareFunc) *Route
	}
)

var (
	methods = [...]string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		PROPFIND,
		http.MethodPut,
		http.MethodTrace,
	}
)

// New creates an instance of Pisces.
func New() (pisces *Pisces) {
	pisces = &Pisces{
		maxParam:  new(int),
		Server:    new(http.Server),
		TLSServer: new(http.Server),
		Logger:    logrus.New(),
	}
	pisces.Server.Handler = pisces
	pisces.TLSServer.Handler = pisces
	pisces.HTTPErrorHandler = pisces.DefaultHTTPErrorHandler
	pisces.Logger.SetLevel(logrus.InfoLevel)
	pisces.StdLogger = log.New(pisces.Logger.Out, "pisces: ", 0)
	pisces.pool.New = func() interface{} {
		return pisces.NewContext(nil, nil)
	}
	pisces.router = NewRouter(pisces)
	return
}

// Context
// NewContext returns a Context instance.
func (pisces *Pisces) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &context{
		request:  r,
		response: NewResponse(w, pisces),
		store:    make(Map),
		pisces:   pisces,
		pvalues:  make([]string, *pisces.maxParam),
		handler:  NotFoundHandler,
	}
}

// AcquireContext returns an empty `Context` instance from the pool.
// You must return the context by calling `ReleaseContext()`.
func (pisces *Pisces) AcquireContext() Context {
	return pisces.pool.Get().(Context)
}

// ReleaseContext returns the `Context` instance back to the pool.
// You must call it after `AcquireContext()`.
func (pisces *Pisces) ReleaseContext(c Context) {
	pisces.pool.Put(c)
}

// MiddlewareFunc
// Pre adds middleware to the chain which is run before router.
func (pisces *Pisces) Pre(middleware ...MiddlewareFunc) {
	pisces.premiddleware = append(pisces.premiddleware, middleware...)
}

// Use adds middleware to the chain which is run after router.
func (pisces *Pisces) Use(middleware ...MiddlewareFunc) {
	pisces.middleware = append(pisces.middleware, middleware...)
}

// Use implements `Pisces#Use()` for sub-routes within the Group.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
	// Allow all requests to reach the group as they might get dropped if router
	// doesn't find a match, making none of the group middleware process.
	for _, p := range []string{"", "/*"} {
		g.pisces.Any(path.Clean(g.prefix+p), func(c Context) error {
			return NotFoundHandler(c)
		}, g.middleware...)
	}
}

// WrapMiddleware wraps `func(http.Handler) http.Handler` into `pisces.MiddlewareFunc`
func WrapMiddleware(m func(http.Handler) http.Handler) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				err = next(c)
			})).ServeHTTP(c.Response(), c.Request())
			return
		}
	}
}

// Route
// Router returns router.
func (pisces *Pisces) Router() *Router {
	return pisces.router
}

// CONNECT registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodConnect, path, h, m...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (pisces *Pisces) DELETE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodDelete, path, h, m...)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (pisces *Pisces) GET(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodGet, path, h, m...)
}

// HEAD registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) HEAD(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodHead, path, h, m...)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodOptions, path, h, m...)
}

// PATCH registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) PATCH(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodPatch, path, h, m...)
}

// POST registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) POST(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodPost, path, h, m...)
}

// PUT registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) PUT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodPut, path, h, m...)
}

// TRACE registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (pisces *Pisces) TRACE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return pisces.Add(http.MethodTrace, path, h, m...)
}

// Any registers a new route for all HTTP methods and path with matching handler
// in the router with optional route-level middleware.
func (pisces *Pisces) Any(path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = pisces.Add(m, path, handler, middleware...)
	}
	return routes
}

// Match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (pisces *Pisces) Match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = pisces.Add(m, path, handler, middleware...)
	}
	return routes
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (pisces *Pisces) Static(prefix, root string) *Route {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}
	return static(pisces, prefix, root)
}

func static(i i, prefix, root string) *Route {
	h := func(c Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}
		name := filepath.Join(root, path.Clean("/"+p)) // "/"+ for security
		return c.File(name)
	}
	i.GET(prefix, h)
	if prefix == "/" {
		return i.GET(prefix+"*", h)
	}

	return i.GET(prefix+"/*", h)
}

// File registers a new route with path to serve a static file with optional route-level middleware.
func (pisces *Pisces) File(path, file string, m ...MiddlewareFunc) *Route {
	return pisces.GET(path, func(c Context) error {
		return c.File(file)
	}, m...)
}

// Add registers a new route for an HTTP method and path with matching handler
// in the router with optional route-level middleware.
func (pisces *Pisces) Add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	name := handlerName(handler)
	pisces.router.Add(method, path, func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i](h)
		}
		return h(c)
	})
	r := &Route{
		Method: method,
		Path:   path,
		Name:   name,
	}
	pisces.router.routes[method+path] = r
	return r
}

// Group creates a new router group with prefix and optional group-level middleware.
func (pisces *Pisces) Group(prefix string, m ...MiddlewareFunc) (g *Group) {
	g = &Group{prefix: prefix, pisces: pisces}
	g.Use(m...)
	return
}

// URI generates a URI from handler.
func (pisces *Pisces) URI(handler HandlerFunc, params ...interface{}) string {
	name := handlerName(handler)
	return pisces.Reverse(name, params...)
}

// URL is an alias for `URI` function.
func (pisces *Pisces) URL(h HandlerFunc, params ...interface{}) string {
	return pisces.URI(h, params...)
}

// Reverse generates an URL from route name and provided parameters.
func (pisces *Pisces) Reverse(name string, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	for _, r := range pisces.router.routes {
		if r.Name == name {
			for i, l := 0, len(r.Path); i < l; i++ {
				if r.Path[i] == ':' && n < ln {
					for ; i < l && r.Path[i] != '/'; i++ {
					}
					uri.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}
				if i < l {
					uri.WriteByte(r.Path[i])
				}
			}
			break
		}
	}
	return uri.String()
}

// Routes returns the registered routes.
func (pisces *Pisces) Routes() []*Route {
	routes := make([]*Route, 0, len(pisces.router.routes))
	for _, v := range pisces.router.routes {
		routes = append(routes, v)
	}
	return routes
}

// Errors
var (
	ErrUnsupportedMediaType        = NewHTTPError(http.StatusUnsupportedMediaType)
	ErrNotFound                    = NewHTTPError(http.StatusNotFound)
	ErrUnauthorized                = NewHTTPError(http.StatusUnauthorized)
	ErrForbidden                   = NewHTTPError(http.StatusForbidden)
	ErrMethodNotAllowed            = NewHTTPError(http.StatusMethodNotAllowed)
	ErrStatusRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge)
	ErrTooManyRequests             = NewHTTPError(http.StatusTooManyRequests)
	ErrBadRequest                  = NewHTTPError(http.StatusBadRequest)
	ErrBadGateway                  = NewHTTPError(http.StatusBadGateway)
	ErrInternalServerError         = NewHTTPError(http.StatusInternalServerError)
	ErrRequestTimeout              = NewHTTPError(http.StatusRequestTimeout)
	ErrServiceUnavailable          = NewHTTPError(http.StatusServiceUnavailable)
	ErrValidatorNotRegistered      = errors.New("validator not registered")
	ErrRendererNotRegistered       = errors.New("renderer not registered")
	ErrInvalidRedirectCode         = errors.New("invalid redirect status code")
	ErrCookieNotFound              = errors.New("cookie not found")
)

// Error handlers
var (
	NotFoundHandler = func(c Context) error {
		return ErrNotFound
	}

	MethodNotAllowedHandler = func(c Context) error {
		return ErrMethodNotAllowed
	}
)

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
}

func (he *HTTPError) SetInternal(err error) *HTTPError {
	he.Internal = err
	return he
}

// DefaultHTTPErrorHandler is the default HTTP error handler. It sends a JSON response
// with status code.
func (pisces *Pisces) DefaultHTTPErrorHandler(err error, c Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
		if he.Internal != nil {
			err = fmt.Errorf("%v, %v", err, he.Internal)
		}
	} else {
		msg = http.StatusText(code)
	}
	if _, ok := msg.(string); ok {
		msg = Map{"message": msg}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead { // Issue #608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, msg)
		}
		if err != nil {
			pisces.Logger.Error(err)
		}
	}
}

// Handler
// WrapHandler wraps `http.Handler` into `pisces.HandlerFunc`.
func WrapHandler(h http.Handler) HandlerFunc {
	return func(c Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

// Server

// ServeHTTP implements `http.Handler` interface, which serves HTTP requests.
func (pisces *Pisces) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
	c := pisces.pool.Get().(*context)
	c.Reset(r, w)

	h := NotFoundHandler

	if pisces.premiddleware == nil {
		pisces.router.Find(r.Method, getPath(r), c)
		h = c.Handler()
		for i := len(pisces.middleware) - 1; i >= 0; i-- {
			h = pisces.middleware[i](h)
		}
	} else {
		h = func(c Context) error {
			pisces.router.Find(r.Method, getPath(r), c)
			h := c.Handler()
			for i := len(pisces.middleware) - 1; i >= 0; i-- {
				h = pisces.middleware[i](h)
			}
			return h(c)
		}
		for i := len(pisces.premiddleware) - 1; i >= 0; i-- {
			h = pisces.premiddleware[i](h)
		}
	}

	// Execute chain
	if err := h(c); err != nil {
		pisces.HTTPErrorHandler(err, c)
	}

	// Release context
	pisces.pool.Put(c)
}

// Start starts an HTTP server.
func (pisces *Pisces) Start(address string) error {
	pisces.Server.Addr = address
	return pisces.StartServer(pisces.Server)
}

// StartTLS starts an HTTPS server.
func (pisces *Pisces) StartTLS(address string, certFile, keyFile string) (err error) {
	if certFile == "" || keyFile == "" {
		return errors.New("invalid tls configuration")
	}
	s := pisces.TLSServer
	s.TLSConfig = new(tls.Config)
	s.TLSConfig.Certificates = make([]tls.Certificate, 1)
	s.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}
	return pisces.startTLS(address)
}

func (pisces *Pisces) startTLS(address string) error {
	s := pisces.TLSServer
	s.Addr = address
	s.TLSConfig.NextProtos = append(s.TLSConfig.NextProtos, "h2")
	return pisces.StartServer(pisces.TLSServer)
}

// StartServer starts a custom http server.
func (pisces *Pisces) StartServer(s *http.Server) (err error) {
	// Setup
	s.ErrorLog = pisces.StdLogger
	s.Handler = pisces

	if s.TLSConfig == nil {
		if pisces.Listener == nil {
			pisces.Listener, err = newListener(s.Addr)
			if err != nil {
				return err
			}
		}
		pisces.Logger.WithField("addr", pisces.Listener.Addr()).Info("http server started on %s")
		return s.Serve(pisces.Listener)
	}
	if pisces.TLSListener == nil {
		l, err := newListener(s.Addr)
		if err != nil {
			return err
		}
		pisces.TLSListener = tls.NewListener(l, s.TLSConfig)
	}
	pisces.Logger.WithField("addr", pisces.TLSListener.Addr()).Info("https server started")
	return s.Serve(pisces.TLSListener)
}

// Close immediately stops the server.
// It internally calls `http.Server#Close()`.
func (pisces *Pisces) Close() error {
	if err := pisces.TLSServer.Close(); err != nil {
		return err
	}
	return pisces.Server.Close()
}

// Shutdown stops server the gracefully.
// It internally calls `http.Server#Shutdown()`.
func (pisces *Pisces) Shutdown(ctx stdContext.Context) error {
	if err := pisces.TLSServer.Shutdown(ctx); err != nil {
		return err
	}
	return pisces.Server.Shutdown(ctx)
}

func getPath(r *http.Request) string {
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	return path
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func newListener(address string) (*tcpKeepAliveListener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &tcpKeepAliveListener{l.(*net.TCPListener)}, nil
}
