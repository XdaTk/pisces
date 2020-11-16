package pisces

import (
	"log"
	"net/http"
	"reflect"
	"runtime"

	"github.com/xdatk/pisces/internal/constant"
)

/************************************/
/*********** HandlerFunc ************/
/************************************/

// HandlerFunc defines a function to serve HTTP requests.
type HandlerFunc func(*Context) error

func notFoundHandler(c *Context) error {
	return ErrNotFound
}

func redirectFixedPathHandler(c *Context) error {
	rPath := c.Path()
	rMethod := c.Method()

	p := rPath

	if length := len(p); length > 1 && p[length-1] == '/' {
		p = p[:length-1]
	} else {
		p = p + "/"
	}

	code := http.StatusMovedPermanently
	if rMethod != http.MethodGet {
		code = http.StatusTemporaryRedirect
	}
	return c.Redirect(code, p)
}

func methodNotAllowedHandler(c *Context) error {
	return c.Text(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

/************************************/
/*********** ErrorHandler ***********/
/************************************/

type ErrorHandler func(*Context, error)

func DefaultErrorHandler(c *Context, err error) {
	if !c.IsCommitted() {
		he, ok := err.(*HTTPError)
		if ok {
			if he.Internal != nil {
				if herr, ok := he.Internal.(*HTTPError); ok {
					he = herr
				}
			}
		} else {
			he = &HTTPError{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			}
		}

		code := he.Code
		message := he.Message

		if c.Method() == http.MethodHead {
			err = c.NoContent(he.Code)
		}

		if c.ContentType() == constant.MIMEApplicationJSON {
			if m, ok := message.(string); ok {
				message = map[string]string{"message": m}
			}
			err = c.JSON(code, message)
		}

		err = c.Text(code, message.(string))
	}
	log.Printf("%v", err)
}

/************************************/
/********** MiddlewareFunc **********/
/************************************/

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
