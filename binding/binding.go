package binding

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Param interface {
	Name() string
	Bind(params map[string][]string, obj interface{}) error
}

type Form interface {
	Name() string
	Bind(values url.Values, obj interface{}) error
}

type Header interface {
	Name() string
	Bind(header http.Header, obj interface{}) error
}

type MultipartFrom interface {
	Name() string
	Bind(form *multipart.Form, obj interface{}) error
}

type Body interface {
	Name() string
	Bind(body io.ReadCloser, obj interface{}) error
}
