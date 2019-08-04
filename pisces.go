package pisces

import (
	"bytes"
	stdContext "context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
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
	// PROPFIND Method can be used on collection and property resources.
	PROPFIND = "PROPFIND"
	// REPORT Method can be used to get information about a resource, see rfc 3253
	REPORT = "REPORT"
)

// MIME types
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
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXXSSProtection                  = "X-XSS-Protection"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderXCSRFToken                      = "X-CSRF-Token"
)

type (
	// Pisces is the top-level framework instance.
	Pisces struct {
		Logger           *logrus.Logger
		Server           *http.Server
		Listener         net.Listener
		HTTPErrorHandler HTTPErrorHandler
		StdLogger        *log.Logger
		pool             sync.Pool
		maxParam         *int
		router           *Router
		routers          map[string]*Router
		notFoundHandler  HandlerFunc
		premiddleware    []MiddlewareFunc
		middleware       []MiddlewareFunc
		common
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Name   string `json:"name"`
	}

	// HTTPError represents an error that occurred while handling a request.
	HTTPError struct {
		Code     int
		Message  interface{}
		Internal error // Stores the error returned by an external dependency
	}

	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(HandlerFunc) HandlerFunc

	// HandlerFunc defines a function to serve HTTP requests.
	HandlerFunc func(Context) error

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// Map defines a generic map of type `map[string]interface{}`.
	Map map[string]interface{}

	// Common struct for pisces & Group.
	common struct{}
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
		REPORT,
	}
)

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
	ErrInvalidCertOrKeyType        = errors.New("invalid cert or key type, must be string or []byte")
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

func New() (p *Pisces) {
	p = &Pisces{}
	p.Logger = logrus.New()
	p.Logger.SetLevel(logrus.InfoLevel)

	p.Server = new(http.Server)
	p.Server.Handler = p
	p.HTTPErrorHandler = p.DefaultHTTPErrorHandler
	p.StdLogger = log.New(p.Logger.Out, "pisces: ", 0)
	p.pool.New = func() interface{} {
		return p.NewContext(nil, nil)
	}
	p.maxParam = new(int)
	p.router = NewRouter(p)
	p.routers = map[string]*Router{}
	return
}

// NewContext returns a Context instance.
func (p *Pisces) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &context{
		request:  r,
		response: NewResponse(w, p),
		store:    make(Map),
		pisces:   p,
		pvalues:  make([]string, *p.maxParam),
		handler:  NotFoundHandler,
	}
}

// Router returns the default router.
func (p *Pisces) Router() *Router {
	return p.router
}

// Routers returns the map of host => router.
func (p *Pisces) Routers() map[string]*Router {
	return p.routers
}

// DefaultHTTPErrorHandler is the default HTTP error handler. It sends a JSON response
// with status code.
func (p *Pisces) DefaultHTTPErrorHandler(err error, c Context) {
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
			p.Logger.Error(err)
		}
	}
}

// Pre adds middleware to the chain which is run before router.
func (p *Pisces) Pre(middleware ...MiddlewareFunc) {
	p.premiddleware = append(p.premiddleware, middleware...)
}

// Use adds middleware to the chain which is run after router.
func (p *Pisces) Use(middleware ...MiddlewareFunc) {
	p.middleware = append(p.middleware, middleware...)
}

// CONNECT registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodConnect, path, h, m...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (p *Pisces) DELETE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodDelete, path, h, m...)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (p *Pisces) GET(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodGet, path, h, m...)
}

// HEAD registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) HEAD(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodHead, path, h, m...)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodOptions, path, h, m...)
}

// PATCH registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) PATCH(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodPatch, path, h, m...)
}

// POST registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) POST(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodPost, path, h, m...)
}

// PUT registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) PUT(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodPut, path, h, m...)
}

// TRACE registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (p *Pisces) TRACE(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return p.Add(http.MethodTrace, path, h, m...)
}

// Any registers a new route for all HTTP methods and path with matching handler
// in the router with optional route-level middleware.
func (p *Pisces) Any(path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = p.Add(m, path, handler, middleware...)
	}
	return routes
}

// Match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (p *Pisces) Match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		routes[i] = p.Add(m, path, handler, middleware...)
	}
	return routes
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (p *Pisces) Static(prefix, root string) *Route {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}
	return p.static(prefix, root, p.GET)
}

func (common) static(prefix, root string, get func(string, HandlerFunc, ...MiddlewareFunc) *Route) *Route {
	h := func(c Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}
		name := filepath.Join(root, path.Clean("/"+p)) // "/"+ for security
		return c.File(name)
	}
	if prefix == "/" {
		return get(prefix+"*", h)
	}
	return get(prefix+"/*", h)
}

func (common) file(path, file string, get func(string, HandlerFunc, ...MiddlewareFunc) *Route,
	m ...MiddlewareFunc) *Route {
	return get(path, func(c Context) error {
		return c.File(file)
	}, m...)
}

// File registers a new route with path to serve a static file with optional route-level middleware.
func (p *Pisces) File(path, file string, m ...MiddlewareFunc) *Route {
	return p.file(path, file, p.GET, m...)
}

func (p *Pisces) add(host, method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	name := handlerName(handler)
	router := p.findRouter(host)
	router.Add(method, path, func(c Context) error {
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
	p.router.routes[method+path] = r
	return r
}

// Add registers a new route for an HTTP method and path with matching handler
// in the router with optional route-level middleware.
func (p *Pisces) Add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	return p.add("", method, path, handler, middleware...)
}

// Host creates a new router group for the provided host and optional host-level middleware.
func (p *Pisces) Host(name string, m ...MiddlewareFunc) (g *Group) {
	p.routers[name] = NewRouter(p)
	g = &Group{host: name, pisces: p}
	g.Use(m...)
	return
}

// Group creates a new router group with prefix and optional group-level middleware.
func (p *Pisces) Group(prefix string, m ...MiddlewareFunc) (g *Group) {
	g = &Group{prefix: prefix, pisces: p}
	g.Use(m...)
	return
}

// URI generates a URI from handler.
func (p *Pisces) URI(handler HandlerFunc, params ...interface{}) string {
	name := handlerName(handler)
	return p.Reverse(name, params...)
}

// URL is an alias for `URI` function.
func (p *Pisces) URL(h HandlerFunc, params ...interface{}) string {
	return p.URI(h, params...)
}

// Reverse generates an URL from route name and provided parameters.
func (p *Pisces) Reverse(name string, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	for _, r := range p.router.routes {
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
func (p *Pisces) Routes() []*Route {
	routes := make([]*Route, 0, len(p.router.routes))
	for _, v := range p.router.routes {
		routes = append(routes, v)
	}
	return routes
}

// AcquireContext returns an empty `Context` instance from the pool.
// You must return the context by calling `ReleaseContext()`.
func (p *Pisces) AcquireContext() Context {
	return p.pool.Get().(Context)
}

// ReleaseContext returns the `Context` instance back to the pool.
// You must call it after `AcquireContext()`.
func (p *Pisces) ReleaseContext(c Context) {
	p.pool.Put(c)
}

// ServeHTTP implements `http.Handler` interface, which serves HTTP requests.
func (p *Pisces) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
	c := p.pool.Get().(*context)
	c.Reset(r, w)

	h := NotFoundHandler

	if p.premiddleware == nil {
		p.findRouter(r.Host).Find(r.Method, getPath(r), c)
		h = c.Handler()
		h = applyMiddleware(h, p.middleware...)
	} else {
		h = func(c Context) error {
			p.findRouter(r.Host).Find(r.Method, getPath(r), c)
			h := c.Handler()
			h = applyMiddleware(h, p.middleware...)
			return h(c)
		}
		h = applyMiddleware(h, p.premiddleware...)
	}

	// Execute chain
	if err := h(c); err != nil {
		p.HTTPErrorHandler(err, c)
	}

	// Release context
	p.pool.Put(c)
}

// Start starts an HTTP server.
func (p *Pisces) Start(address string) error {
	p.Server.Addr = address
	return p.StartServer(p.Server)
}

func filepathOrContent(fileOrContent interface{}) (content []byte, err error) {
	switch v := fileOrContent.(type) {
	case string:
		return ioutil.ReadFile(v)
	case []byte:
		return v, nil
	default:
		return nil, ErrInvalidCertOrKeyType
	}
}

// StartServer starts a custom http server.
func (p *Pisces) StartServer(s *http.Server) (err error) {
	s.ErrorLog = p.StdLogger
	s.Handler = p
	if p.Listener == nil {
		p.Listener, err = newListener(s.Addr)
		if err != nil {
			return err
		}
	}
	p.Logger.WithField("addr", p.Listener.Addr()).Info("http server started")
	return s.Serve(p.Listener)
}

// Close immediately stops the server.
// It internally calls `http.Server#Close()`.
func (p *Pisces) Close() error {
	return p.Server.Close()
}

// Shutdown stops the server gracefully.
// It internally calls `http.Server#Shutdown()`.
func (p *Pisces) Shutdown(ctx stdContext.Context) error {
	return p.Server.Shutdown(ctx)
}

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

// SetInternal sets error to HTTPError.Internal
func (he *HTTPError) SetInternal(err error) *HTTPError {
	he.Internal = err
	return he
}

// WrapHandler wraps `http.Handler` into `pisces.HandlerFunc`.
func WrapHandler(h http.Handler) HandlerFunc {
	return func(c Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
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

func getPath(r *http.Request) string {
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	return path
}

func (p *Pisces) findRouter(host string) *Router {
	if len(p.routers) > 0 {
		if r, ok := p.routers[host]; ok {
			return r
		}
	}
	return p.router
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

// // PathUnescape is wraps `url.PathUnescape`
// func PathUnescape(s string) (string, error) {
// 	return url.PathUnescape(s)
// }

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	if c, err = ln.AcceptTCP(); err != nil {
		return
	} else if err = c.(*net.TCPConn).SetKeepAlive(true); err != nil {
		return
	} else if err = c.(*net.TCPConn).SetKeepAlivePeriod(3 * time.Minute); err != nil {
		return
	}
	return
}

func newListener(address string) (*tcpKeepAliveListener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &tcpKeepAliveListener{l.(*net.TCPListener)}, nil
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
