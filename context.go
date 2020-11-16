package pisces

import (
	"fmt"
	"github.com/xdatk/pisces/binding"
	"github.com/xdatk/pisces/internal/constant"
	"github.com/xdatk/pisces/render"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Context struct {
	engine *Engine

	writerMem responseWriter
	paramsMem *Params

	Request *http.Request
	Writer  ResponseWriter

	params      Params
	fullPath    string
	handlerName string
	handler     HandlerFunc

	queryCache              url.Values
	headerCache             http.Header
	cookieCache             []*http.Cookie
	parseFrom               *bool
	parseFromError          error
	parseMultipartForm      *bool
	parseMultipartFormError error
}

func (c *Context) reset(writer http.ResponseWriter, request *http.Request) {
	c.writerMem.reset(writer)
	*c.paramsMem = (*c.paramsMem)[0:0]

	c.Request = request
	c.Writer = &c.writerMem

	c.params = c.params[0:0]
	c.fullPath = ""
	c.handlerName = ""
	c.handler = nil

	c.queryCache = nil
	c.cookieCache = nil
	c.parseFrom = nil
	c.parseFromError = nil
	c.parseMultipartForm = nil
	c.parseMultipartFormError = nil
}

/************************************/
/************ Input Data ************/
/************************************/

// HandlerName returns the main handler's name. For example if the handler is "handleGetUsers()",
// this function will return "main.handleGetUsers".
func (c *Context) HandlerName() string {
	return c.handlerName
}

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc {
	return c.handler
}

// Path returns the url path of the request.
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// Method returns the method of the request.
func (c *Context) Method() string {
	return c.Request.Method
}

// IsTLS returns true if HTTP connection is TLS otherwise false.
func (c *Context) IsTLS() bool {
	return c.Request.TLS != nil
}

// Param returns the keyed url param. if it exists.
// otherwise it returns an empty string `("")`.
func (c *Context) Param(key string) string {
	return c.GetParams().ByName(key)
}

// DefaultParam returns the keyed url param if it exists,
// otherwise it returns the specified defaultValue string.
func (c *Context) DefaultParam(key, defaultValue string) string {
	k, ok := c.GetParam(key)
	if ok {
		return k
	}
	return defaultValue
}

// GetParam is like Param(), it returns the keyed url param
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
func (c *Context) GetParam(key string) (string, bool) {
	return c.GetParams().Get(key)
}

// Params returns URI params.
func (c *Context) GetParams() Params {
	return c.params
}

// Header returns the keyed header value if it exists,
// otherwise it returns an empty string `("")`.
func (c *Context) Header(key string) string {
	return c.GetHeaders().Get(key)
}

// DefaultHeader returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
func (c *Context) DefaultHeader(key, defaultValue string) string {
	if value, ok := c.GetHeader(key); ok {
		return value
	}
	return defaultValue
}

// GetHeader is like Header(), it returns the keyed header value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
func (c *Context) GetHeader(key string) (string, bool) {
	_, ok := c.GetHeaders()[key]
	return c.Header(key), ok
}

// HeaderValues returns the keyed header values if it exists,
// otherwise it returns an empty string `("")`.
func (c *Context) HeaderValues(key string) []string {
	return c.GetHeaders().Values(key)
}

// GetHeaderValues is like HeaderValues(), it returns the keyed header values
// if it exists `(value, true)` (even when the value is an empty string slice),
// otherwise it returns `("", false)`.
func (c *Context) GetHeaderValues(key string) ([]string, bool) {
	_, ok := c.GetHeaders()[key]
	return c.HeaderValues(key), ok
}

// GetCookies returns the cookies of the request.
func (c *Context) GetCookies() []*http.Cookie {
	if c.cookieCache == nil {
		c.cookieCache = c.Request.Cookies()
	}

	return c.cookieCache
}

// GetHeaders returns the header of the request.
func (c *Context) GetHeaders() http.Header {
	return c.Request.Header
}

// Scheme returns the HTTP protocol scheme, `http` or `https`.
func (c *Context) Scheme() string {
	if c.IsTLS() {
		return "https"
	}
	if scheme := c.Header(constant.HeaderXForwardedProto); scheme != "" {
		return scheme
	}
	if scheme := c.Header(constant.HeaderXForwardedProtocol); scheme != "" {
		return scheme
	}
	if ssl := c.Header(constant.HeaderXForwardedSsl); ssl == "on" {
		return "https"
	}
	if scheme := c.Header(constant.HeaderXUrlScheme); scheme != "" {
		return scheme
	}
	return "http"
}

// ContentType returns the Content-Type header of the request.
func (c *Context) ContentType() string {
	s := c.Header(constant.HeaderContentType)
	for i, char := range s {
		if char == ' ' || char == ';' {
			return s[:i]
		}
	}
	return s
}

// UserAgent returns the User-Agent header of the request.
func (c *Context) UserAgent() string {
	return c.Request.UserAgent()
}

// IsWebsocket returns true if the request headers indicate that a websocket
// handshake is being initiated by the client.
func (c *Context) IsWebSocket() bool {
	upgrade := c.Header(constant.HeaderUpgrade)
	return strings.ToLower(upgrade) == "websocket"
}

// RealIP returns the client's network address
func (c *Context) RealIP() string {
	if ip := c.Header(constant.HeaderXForwardedFor); ip != "" {
		return strings.Split(ip, ", ")[0]
	}
	if ip := c.Header(constant.HeaderXRealIP); ip != "" {
		return ip
	}
	ra, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ra
}

// GetHeaders returns the header of the request.
func (c *Context) initQueueCache() {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// DefaultQuery returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// GetQuery is like Query(), it returns the keyed url query value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// QueryArray returns a slice of strings for a given query key.
func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

// GetQueryArray returns a slice of strings for a given query key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	q, v := c.GetQuerys()[key]
	return q, v
}

// GetQuerys returns the query params of the request.
func (c *Context) GetQuerys() url.Values {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}

	return c.queryCache
}

// GetPostForm returns value for a given form key.
func (c *Context) Form(key string) (string, error) {
	value, _, err := c.GetForm(key)
	return value, err
}

// GetForm returns value for a given form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetForm(key string) (string, bool, error) {
	values, ok, err := c.GetFormArray(key)
	if err != nil {
		return "", false, err
	}
	if ok {
		return values[0], ok, err
	}
	return "", false, err
}

// FormArray returns a slice of strings for a given form key.
func (c *Context) FormArray(key string) ([]string, error) {
	values, _, err := c.GetFormArray(key)
	return values, err
}

// GetFormArray returns a slice of strings for a given form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetFormArray(key string) ([]string, bool, error) {
	values, err := c.GetForms()
	if err != nil {
		return []string{}, false, c.parseFromError
	}

	if values := values[key]; len(values) > 0 {
		return values, true, nil
	}
	return []string{}, false, nil
}

// GetForms returns post from params
func (c *Context) GetForms() (url.Values, error) {
	c.initPostFormCache()

	if *c.parseFrom {
		return c.Request.Form, c.parseFromError
	} else {
		return nil, c.parseFromError
	}
}

// GetPostForm returns value for a given post form key.
func (c *Context) PostForm(key string) (string, error) {
	value, _, err := c.GetPostForm(key)
	return value, err
}

// GetPostForm returns value for a given post form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetPostForm(key string) (string, bool, error) {
	values, ok, err := c.GetPostFormArray(key)
	if err != nil {
		return "", false, err
	}
	if ok {
		return values[0], ok, err
	}
	return "", false, err
}

// PostFormArray returns a slice of strings for a given post form key.
func (c *Context) PostFormArray(key string) ([]string, error) {
	values, _, err := c.GetPostFormArray(key)
	return values, err
}

// GetPostFormArray returns a slice of strings for a given post form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetPostFormArray(key string) ([]string, bool, error) {
	values, err := c.GetPostForms()
	if err != nil {
		return []string{}, false, c.parseFromError
	}

	if values := values[key]; len(values) > 0 {
		return values, true, nil
	}
	return []string{}, false, nil
}

func (c *Context) initPostFormCache() {
	if c.parseFrom == nil {
		b := false
		err := c.Request.ParseForm()
		if err != nil {
			c.parseFromError = err
		} else {
			b = true
		}

		c.parseFrom = &b
	}
}

// GetPostForms returns post from params
func (c *Context) GetPostForms() (url.Values, error) {
	c.initPostFormCache()

	if *c.parseFrom {
		return c.Request.PostForm, c.parseFromError
	} else {
		return nil, c.parseFromError
	}
}

// FormFile returns the first file for the provided form key.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	_, err := c.GetMultipartForms()
	if err != nil {
		return nil, err
	}

	f, fh, err := c.Request.FormFile(name)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return fh, err
}

// GetMultipartForms is the parsed multipart form, including file uploads.
func (c *Context) GetMultipartForms() (*multipart.Form, error) {
	if c.parseMultipartForm == nil {
		b := false
		err := c.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			c.parseMultipartFormError = err
		} else {
			b = true
			c.parseFrom = &b
		}

		c.parseMultipartForm = &b
	}

	return c.Request.MultipartForm, c.parseMultipartFormError
}

// Body returns request body
func (c *Context) Body() io.ReadCloser {
	return c.Request.Body
}

// Bind checks the Content-Type to select a binding engine automatically,
// support from, body binding, otherwise returns an error.
func (c *Context) Bind(obj interface{}) error {
	return c.engine.binder.Bind(c, obj)
}

// BindParam is binding request url params to obj.
func (c *Context) BindParam(obj interface{}) error {
	m := make(map[string][]string)
	for _, v := range c.params {
		m[v.Key] = []string{v.Value}
	}
	return c.engine.binder.Param.Bind(m, obj)
}

// BindQuery is binding request query to obj.
func (c *Context) BindQuery(obj interface{}) error {
	return c.engine.binder.Form.Bind(c.GetQuerys(), obj)
}

// BindFormUseCustom is binding request form to obj use custom BindingForm.
func (c *Context) BindQueryUseCustom(binder binding.BindingForm, obj interface{}) error {
	return binder.Bind(c.GetQuerys(), obj)
}

// BindHeader is binding request header to obj.
func (c *Context) BindHeader(obj interface{}) error {
	return c.engine.binder.Header.Bind(c.GetHeaders(), obj)
}

// BindFormUseCustom is binding request form to obj use custom BindingHeader.
func (c *Context) BindHeaderUseCustom(binder binding.BindingHeader, obj interface{}) error {
	return binder.Bind(c.GetHeaders(), obj)
}

// BindForm is binding request from to obj.
func (c *Context) BindForm(obj interface{}) error {
	froms, err := c.GetForms()
	if err != nil {
		return err
	}
	return c.engine.binder.Form.Bind(froms, obj)
}

// BindFormUseCustom is binding request form to obj use custom BindingForm.
func (c *Context) BindFormUseCustom(binder binding.BindingForm, obj interface{}) error {
	form, err := c.GetForms()
	if err != nil {
		return err
	}
	return binder.Bind(form, obj)
}

// BindPostForm is binding request post form to obj.
func (c *Context) BindPostForm(obj interface{}) error {
	froms, err := c.GetPostForms()
	if err != nil {
		return err
	}
	return c.engine.binder.PostForm.Bind(froms, obj)
}

// BindPostForm is binding request post form to obj use custom BindingForm.
func (c *Context) BindPostFormUseCustom(binder binding.BindingForm, obj interface{}) error {
	form, err := c.GetPostForms()
	if err != nil {
		return err
	}
	return binder.Bind(form, obj)
}

// BindMultipartFrom is binding request multipart form to obj.
func (c *Context) BindMultipartFrom(obj interface{}) error {
	form, err := c.GetMultipartForms()
	if err != nil {
		return err
	}
	return c.engine.binder.MultipartFrom.Bind(form, obj)
}

// BindMultipartFromUseCustom is binding request multipart form to obj use custom BindingMultipartFrom.
func (c *Context) BindMultipartFromUseCustom(binder binding.BindingMultipartFrom, obj interface{}) error {
	form, err := c.GetMultipartForms()
	if err != nil {
		return err
	}
	return binder.Bind(form, obj)
}

// BindParam is binding request body to obj.
func (c *Context) BindBody(obj interface{}) error {
	binder, ok := c.engine.binder.Body[c.ContentType()]
	if !ok {
		return fmt.Errorf("not support content type")
	}

	return binder.Bind(c.Body(), obj)
}

// BindBodyUseCustom is binding request body to obj use custom bindingBody.
func (c *Context) BindBodyUseCustom(binder binding.BindingBody, obj interface{}) error {
	return binder.Bind(c.Body(), obj)
}

// Validate is validate obj struct
func (c *Context) Validate(obj interface{}) error {
	return c.engine.validator.Validate(obj)
}

/************************************/
/******** RESPONSE RENDERING ********/
/************************************/

func (c *Context) setStatus(code int) {
	c.writerMem.Status = code
}

func (c *Context) IsCommitted() bool {
	return c.writerMem.Committed
}

// SetCookie adds a Set-Cookie header to the ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be
// silently dropped.
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// SetHeader is set header to the response.
// If value == "", this method removes the cookie
func (c *Context) SetHeader(key, value string) {
	if value == "" {
		c.Writer.Header().Del(key)
		return
	}
	c.Writer.Header().Set(key, value)
}

// setContentType is set content-type header to the response.
// If value == "", this method removes the cookie
func (c *Context) setContentType(value string) {
	c.SetHeader(constant.HeaderContentType, value)
}

// NoContent sends a response with no body and a status code.
func (c *Context) NoContent(code int) error {
	c.Writer.WriteHeader(code)
	return nil
}

func (c *Context) renderCodeAndContentType(code int, contentType string) error {
	if (code >= 100 && code <= 199) || code == http.StatusNoContent || code == http.StatusNotModified {
		return fmt.Errorf("unsupport")
	}

	c.setContentType(contentType)
	c.setStatus(code)
	return nil
}

// Render writes the response headers and calls Render to render data.
func (c *Context) Render(code int, r render.Render) (err error) {
	err =c.renderCodeAndContentType(code, r.ContentType())
	if err != nil {
		return err
	}
	body, err := r.Render()
	if err != nil {
		return err
	}
	_, err = c.Writer.Write(body)
	return
}

// JSON serializes the given struct as JSON into the response body.
// It also sets the Content-Type as "application/json".
func (c *Context) JSON(code int, obj interface{}) error {
	return c.Render(code, render.JSON{Data: obj})
}

// Text writes the given string into the response body.
func (c *Context) Text(code int, format string, values ...interface{}) error {
	return c.Render(code, render.Text{Format: format, Data: values})
}

// Data writes some data into the body stream and updates the HTTP code.
func (c *Context) Data(code int, contentType string, data []byte) error {
	return c.Render(code, render.Data{
		Type: contentType,
		Data: data,
	})
}

func (c *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	err = c.renderCodeAndContentType(code, contentType)
	if err != nil {
		return err
	}
	_, err = io.Copy(c.Writer, r)
	return
}

// File writes the specified file into the body stream in a efficient way.
func (c *Context) File(file string) error {
	f, err := os.OpenFile(file, os.O_RDONLY, 4)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f)
	return nil
}

// FileFromFS writes the specified file from http.FileSystem into the body stream in an efficient way.
func (c *Context) FileFromFS(file string, filesystem http.FileSystem) error {
	f, err := filesystem.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f)
	return nil
}

// Attachment writes the specified file into the body stream in an efficient way
// On the client side, the file will typically be downloaded with the given filename
func (c *Context) Attachment(filepath, filename string) error {
	return c.contentDisposition(filepath, filename, "attachment")
}

// Inline writes the specified file into the body stream in an efficient way
// On the client side, the file will typically be open with the given filename
func (c *Context) Inline(filepath, filename string) error {
	return c.contentDisposition(filepath, filename, "inline")
}

func (c *Context) contentDisposition(file, name, dispositionType string) error {
	c.SetHeader(constant.HeaderContentDisposition, fmt.Sprintf("%s; filename=%q", dispositionType, name))
	return c.File(file)
}

// Redirect returns a HTTP redirect to the specific location.
func (c *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return fmt.Errorf("invalid redirect status code")
	}
	c.SetHeader(constant.HeaderLocation, url)
	c.Writer.WriteHeader(code)
	return nil
}

func (c *Context) Error(err error) {
	c.engine.errorHandler(c, err)
}
