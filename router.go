package pisces

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/xdatk/pisces/internal/bytesconv"
	"github.com/xdatk/pisces/internal/util"
)

const (
	indexPage = "index.html"
)

var (
	methods = [...]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
)

/************************************/
/******* Router Specification *******/
/************************************/

// RouteInfo represents a request route's specification which contains method and path and its handler.
type RouteInfo struct {
	Name    string
	Method  string
	Path    string
	Handler HandlerFunc
}

// RoutesInfo defines a RouteInfo array.
type RoutesInfo []RouteInfo

/************************************/
/***** Routes Handler Interface *****/
/************************************/

//Routes defines all router handle interface.
type Routes interface {
	// Use adds middleware to the group, see example code in GitHub.
	Use(...MiddlewareFunc)

	// GET registers a new get request handle and middleware with the given path and method.
	GET(string, HandlerFunc, ...MiddlewareFunc) Routes
	// HEAD registers a new head request handle and middleware with the given path and method.
	HEAD(string, HandlerFunc, ...MiddlewareFunc) Routes
	// POST registers a new post request handle and middleware with the given path and method.
	POST(string, HandlerFunc, ...MiddlewareFunc) Routes
	// PUT registers a new put request handle and middleware with the given path and method.
	PUT(string, HandlerFunc, ...MiddlewareFunc) Routes
	// PATCH registers a new patch request handle and middleware with the given path and method.
	PATCH(string, HandlerFunc, ...MiddlewareFunc) Routes
	// DELETE registers a new delete request handle and middleware with the given path and method.
	DELETE(string, HandlerFunc, ...MiddlewareFunc) Routes
	// CONNECT registers a new connect request handle and middleware with the given path and method.
	CONNECT(string, HandlerFunc, ...MiddlewareFunc) Routes
	// OPTIONS registers a new options request handle and middleware with the given path and method.
	OPTIONS(string, HandlerFunc, ...MiddlewareFunc) Routes
	// TRACE registers a new trace request handle and middleware with the given path and method.
	TRACE(string, HandlerFunc, ...MiddlewareFunc) Routes
	// Any registers a route that matches all the HTTP methods.
	Any(string, HandlerFunc, ...MiddlewareFunc) Routes

	// StaticFile registers a single route in order to serve a single file of the local filesystem.
	StaticFile(string, string, ...MiddlewareFunc) Routes
	// Static serves files from the given file system root.
	Static(string, string, ...MiddlewareFunc) Routes
	// StaticFS works just like `Static()` but a custom `http.FileSystem` can be used instead.
	StaticFS(string, http.FileSystem, ...MiddlewareFunc) Routes
}

/************************************/
/********* Router Interface *********/
/************************************/

// Router defines all router handle interface includes single and group router.
type Router interface {
	Routes
	Group(string, ...MiddlewareFunc) *RouterGroup
}

/************************************/
/*********** Router Group ***********/
/************************************/

// RouterGroup is used internally to configure router, a RouterGroup is associated with
// a prefix and an array of handlers (middleware).
type RouterGroup struct {
	root        bool
	basePath    string
	middlewares []MiddlewareFunc
	engine      *Engine
}

func (group *RouterGroup) Use(middlewares ...MiddlewareFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (group *RouterGroup) addHandler(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	m := make([]MiddlewareFunc, 0, len(group.middlewares)+len(middlewares))
	m = append(m, group.middlewares...)
	m = append(m, middlewares...)
	group.engine.addRouter(method, path, handler, m...)
	return group.returnRoutes()
}

func (group *RouterGroup) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodGet, path, handler, middlewares...)
}

func (group *RouterGroup) HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodHead, path, handler, middlewares...)
}

func (group *RouterGroup) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodPost, path, handler, middlewares...)
}

func (group *RouterGroup) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodPut, path, handler, middlewares...)
}

func (group *RouterGroup) PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodPatch, path, handler, middlewares...)
}

func (group *RouterGroup) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodDelete, path, handler, middlewares...)
}

func (group *RouterGroup) CONNECT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodConnect, path, handler, middlewares...)
}

func (group *RouterGroup) OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodOptions, path, handler, middlewares...)
}

func (group *RouterGroup) TRACE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	return group.addHandler(http.MethodTrace, path, handler, middlewares...)
}

func (group *RouterGroup) Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Routes {
	for _, m := range methods {
		group.addHandler(m, path, handler, middlewares...)
	}
	return group.returnRoutes()
}

func (group *RouterGroup) StaticFile(relativePath string, filepath string, middlewares ...MiddlewareFunc) Routes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static file")
	}

	handler := func(c *Context) error {
		return c.File(filepath)
	}

	group.GET(relativePath, handler, middlewares...)
	group.HEAD(relativePath, handler, middlewares...)

	return group.returnRoutes()
}

func (group *RouterGroup) Static(relativePath, root string, middlewares ...MiddlewareFunc) Routes {
	return group.StaticFS(relativePath, http.Dir(root), middlewares...)
}

func (group *RouterGroup) StaticFS(relativePath string, filesystem http.FileSystem, middlewares ...MiddlewareFunc) Routes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}

	handler := func(c *Context) error {
		file := c.Param("filepath")

		f, err := filesystem.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return err
		}

		if fi.IsDir() {
			file = filepath.Join(file, indexPage)
			f, err = filesystem.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			fi, err = f.Stat()
			if err != nil {
				return err
			}
		}

		http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f)
		return nil
	}

	urlPattern := path.Join(relativePath, "/*filepath")

	group.GET(urlPattern, handler, middlewares...)
	group.HEAD(urlPattern, handler, middlewares...)

	return group.returnRoutes()
}

func (group *RouterGroup) Group(path string, middlewares ...MiddlewareFunc) *RouterGroup {
	return &RouterGroup{
		basePath:    util.JoinPaths(group.basePath, path),
		middlewares: append(group.middlewares, middlewares...),
		engine:      group.engine,
	}
}

func (group *RouterGroup) returnRoutes() Routes {
	if group.root {
		return group.engine
	}
	return group
}

/************************************/
/**************** Param *************/
/************************************/

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// Get returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) Get(name string) (string, bool) {
	for _, entry := range ps {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return "", false
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) (va string) {
	va, _ = ps.Get(name)
	return
}

/************************************/
/************* method tree **********/
/************************************/

type methodTree struct {
	method string
	root   *node
}

type methodTrees []methodTree

func (trees methodTrees) get(method string) *node {
	for _, tree := range trees {
		if tree.method == method {
			return tree.root
		}
	}
	return nil
}

/************************************/
/************* radix tree ***********/
/************************************/

type nodeKind uint8

const (
	skind nodeKind = iota
	rkind
	pkind
	akind
)

type node struct {
	kind      nodeKind
	wildChild bool
	priority  uint32
	indices   string
	path      string
	fullPath  string
	name      string
	handler   HandlerFunc
	children  []*node
}

type nodeValue struct {
	fullPath string
	tsr      bool
	params   *Params
	name     string
	handler  HandlerFunc
}

func (n *node) insert(path, name string, handler HandlerFunc) {
	fullPath := path
	n.priority++

	if n.path == "" && n.indices == "" {
		n.insertChildren(path, fullPath, name, handler)
		n.kind = rkind
		return
	}

	parentFullPathIndex := 0

walk:
	for {
		i := util.LongestCommonPrefix(path, n.path)

		if i < len(n.path) {
			child := node{
				wildChild: n.wildChild,
				priority:  n.priority - 1,
				indices:   n.indices,
				path:      n.path[i:],
				fullPath:  n.fullPath,
				name:      n.name,
				handler:   n.handler,
				children:  n.children,
			}

			n.children = []*node{&child}

			n.wildChild = false
			n.indices = bytesconv.BytesToString([]byte{n.path[i]})
			n.path = path[:i]
			n.fullPath = fullPath[:parentFullPathIndex+i]
			n.name = ""
			n.handler = nil
		}

		if i < len(path) {
			path = path[i:]

			if n.wildChild {
				parentFullPathIndex += len(n.path)
				n = n.children[0]
				n.priority++

				if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
					n.kind != akind &&
					(len(n.path) >= len(path) || path[len(n.path)] == '/') {
					continue walk
				} else {
					pathSeg := path
					if n.kind != akind {
						pathSeg = strings.SplitN(path, "/", 2)[0]
					}
					prefix := fullPath[:strings.Index(fullPath, pathSeg)] + n.path
					panic("'" + pathSeg +
						"' in new path '" + fullPath +
						"' conflicts with existing wildcard '" + n.path +
						"' in existing prefix '" + prefix +
						"'")
				}
			}

			idxc := path[0]

			if n.kind == pkind && idxc == '/' && len(n.children) == 1 {
				parentFullPathIndex += len(n.path)
				n.priority++
				n = n.children[0]
				continue walk
			}

			for i, c := range []byte(n.indices) {
				if c == idxc {
					parentFullPathIndex += len(n.path)
					i = n.incrementChildrenPriority(i)
					n = n.children[i]
					continue walk
				}
			}

			if idxc != ':' && idxc != '*' {
				n.indices += bytesconv.BytesToString([]byte{idxc})
				child := &node{
					fullPath: fullPath,
				}
				n.children = append(n.children, child)
				n.incrementChildrenPriority(len(n.indices) - 1)
				n = child
			}

			n.insertChildren(path, fullPath, name, handler)
			return
		}

		if n.handler != nil {
			panic("handler are already registered for path '" + fullPath + "'")
		}

		n.handler = handler
		n.name = name
		n.fullPath = fullPath
		return
	}
}

func (n *node) insertChildren(path, fullPath, name string, handler HandlerFunc) {
	for {
		wildcard, i, valid := findWildcard(path)

		if i < 0 {
			break
		}

		if !valid {
			panic("only one wildcard per path segment is allowed, has: '" +
				wildcard + "' in path '" + fullPath + "'")
		}

		if len(wildcard) < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if len(n.children) > 0 {
			panic("wildcard segment '" + wildcard +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		if wildcard[0] == ':' {
			if i > 0 {
				n.path = path[:i]
				path = path[i:]
			}

			n.wildChild = true

			child := &node{
				kind:     pkind,
				fullPath: fullPath,
				path:     wildcard,
			}

			n.children = []*node{child}
			n = child
			n.priority++

			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &node{
					priority: 1,
					fullPath: fullPath,
				}
				n.children = []*node{child}
				n = child
				continue
			}

			n.name = name
			n.handler = handler
			return
		}

		if i+len(wildcard) != len(path) {
			panic("akind routes are only allowed at the end of the path in path '" + fullPath + "'")
		}

		i--
		if path[i] != '/' {
			panic("no / before akind in path '" + fullPath + "'")
		}

		n.path = path[:i]
		child := &node{
			kind:      akind,
			wildChild: true,
			fullPath:  fullPath,
		}
		n.children = []*node{child}
		n.indices = string('/')
		n = child
		n.priority++

		child = &node{
			kind:     akind,
			priority: 1,
			path:     path[i:],
			fullPath: fullPath,
			name:     name,
			handler:  handler,
		}

		n.children = []*node{child}
		return
	}

	n.path = path
	n.name = name
	n.handler = handler
	n.fullPath = fullPath
}

func (n *node) find(path string, params *Params) (value nodeValue) {
walk:
	for {
		prefix := n.path
		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				if !n.wildChild {
					idxc := path[0]
					for i, c := range []byte(n.indices) {
						if c == idxc {
							n = n.children[i]
							continue walk
						}
					}

					value.tsr = path == "/" && n.handler != nil
					return
				}

				n = n.children[0]
				switch n.kind {
				case pkind:
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					if params != nil {
						if value.params == nil {
							value.params = params
						}

						i := len(*value.params)
						*value.params = (*value.params)[:i+1]
						val := path[:end]
						(*value.params)[i] = Param{
							Key:   n.path[1:],
							Value: val,
						}
					}

					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						value.tsr = len(path) == end+1
						return
					}

					if value.handler = n.handler; value.handler != nil {
						value.name = n.name
						value.fullPath = n.fullPath
						return
					} else if len(n.children) == 1 {
						n = n.children[0]
						value.tsr = (n.path == "/" && n.handler != nil) || (n.path == "" && n.indices == "/")
					}
					return
				case akind:
					if params != nil {
						if value.params == nil {
							value.params = params
						}
						i := len(*value.params)
						*value.params = (*value.params)[:i+1]
						val := path
						(*value.params)[i] = Param{
							Key:   n.path[2:],
							Value: val,
						}
					}

					value.name = n.name
					value.handler = n.handler
					value.fullPath = n.fullPath
					return
				default:
					panic("invalid node type")
				}

			}
		} else if path == prefix {
			if value.handler = n.handler; value.handler != nil {
				value.name = n.name
				value.fullPath = n.fullPath
				return
			}

			if path == "/" && n.wildChild && n.kind != rkind {
				value.tsr = true
				return
			}

			for i, c := range []byte(n.indices) {
				if c == '/' {
					n = n.children[i]
					value.tsr = (len(n.path) == 1 && n.handler != nil) ||
						(n.kind == akind && n.children[0].handler != nil)
					return
				}
			}

			return
		}

		value.tsr = (path == "/") ||
			(len(prefix) == len(path)+1 && prefix[len(path)] == '/' &&
				path == prefix[:len(prefix)-1] && n.handler != nil)
		return
	}
}

func findWildcard(path string) (wilcard string, i int, valid bool) {
	for start, c := range []byte(path) {
		if c != ':' && c != '*' {
			continue
		}

		valid = true
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*':
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}

func (n *node) incrementChildrenPriority(pos int) int {
	cs := n.children
	cs[pos].priority++
	prio := cs[pos].priority

	newPos := pos
	for ; newPos > 0 && cs[newPos-1].priority < prio; newPos-- {
		cs[newPos-1], cs[newPos] = cs[newPos], cs[newPos-1]
	}

	if newPos != pos {
		n.indices = n.indices[:newPos] +
			n.indices[pos:pos+1] +
			n.indices[newPos:pos] + n.indices[pos+1:]
	}

	return newPos
}
