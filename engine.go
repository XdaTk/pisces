package pisces

import (
	"net/http"
	"sync"

	"github.com/xdatk/pisces/internal/util"
)

// Engine is the framework instance.
// it contains the muxer, middleware and configuration.
type Engine struct {
	RouterGroup

	notFoundHandler HandlerFunc
	errorHandler    ErrorHandler

	binder    Binder
	validator Validator

	pool      sync.Pool
	trees     methodTrees
	maxParams uint16
}

func New() *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			root:     true,
			basePath: "/",
		},
		notFoundHandler: notFoundHandler,
		errorHandler:    DefaultErrorHandler,
		trees:           make(methodTrees, 0, 9),
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

func (e *Engine) NoRoute(handler HandlerFunc) {
	e.notFoundHandler = handler
}

func (e *Engine) allocateContext() *Context {
	v := make(Params, 0, e.maxParams)
	return &Context{engine: e, paramsMem: &v}
}

/************************************/
/*************** Router *************/
/************************************/

func (e *Engine) Routers() (routes RoutesInfo) {
	for _, tree := range e.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
}

func iterate(path string, method string, routes RoutesInfo, root *node) RoutesInfo {
	path += root.path

	if root.handler != nil {
		routes = append(routes, RouteInfo{
			Name:    root.name,
			Method:  method,
			Path:    path,
			Handler: root.handler,
		})
	}

	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}

	return routes
}

func (e *Engine) addRouter(method string, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	if method == "" {
		panic("method must not be empty")
	}

	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if handler == nil {
		panic("handler must not be nil")
	}

	root := e.trees.get(method)

	if root == nil {
		root = new(node)
		root.fullPath = "/"
		e.engine.trees = append(e.trees, methodTree{method: method, root: root})
	}

	name := handlerName(handler)

	root.insert(path, name, func(c *Context) error {
		h := applyMiddleware(handler, middlewares...)
		return h(c)
	})

	if paramsCount := util.CountParams(path); paramsCount > e.maxParams {
		e.maxParams = paramsCount
	}
}

/************************************/
/*********** http.Handler ***********/
/************************************/

func (e *Engine) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	c := e.pool.Get().(*Context)
	c.reset(writer, request)

	e.handleHTTPRequest(c)

	e.pool.Put(c)
}

func (e *Engine) handleHTTPRequest(c *Context) {
	rMethod := c.Request.Method
	rPath := c.Request.URL.Path

	t := e.trees
	var root *node

	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != rMethod {
			continue
		}

		root = t[i].root
	}

	if root != nil {
		value := root.find(rPath, c.paramsMem)

		if value.params != nil {
			c.params = *value.params
		}

		c.fullPath = value.fullPath
		c.handlerName = value.name

		if value.handler != nil {
			c.handler = value.handler
		} else if rMethod != http.MethodConnect && rPath != "/" && value.tsr {
			c.handler = redirectFixedPathHandler
		}
	} else {
		for _, tree := range e.trees {
			if tree.method == rMethod {
				continue
			}

			tv := tree.root.find(rPath, nil)

			if tv.handler != nil {
				c.handler = methodNotAllowedHandler
				break
			}
		}
	}

	if c.handler == nil {
		c.handler = notFoundHandler
	}

	err := c.handler(c)
	if err != nil {
		e.errorHandler(c, err)
	}
}
