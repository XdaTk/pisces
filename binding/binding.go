package binding

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type BindingParam interface {
	Name() string
	Bind(params map[string][]string, obj interface{}) error
}

type BindingForm interface {
	Name() string
	Bind(values url.Values, obj interface{}) error
}

type BindingHeader interface {
	Name() string
	Bind(header http.Header, obj interface{}) error
}

type BindingMultipartFrom interface {
	Name() string
	Bind(form *multipart.Form, obj interface{}) error
}

type BindingBody interface {
	Name() string
	Bind(body io.ReadCloser, obj interface{}) error
}
