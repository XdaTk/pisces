package binding

import (
	"net/http"
)

type HeaderBinding struct {
}

func (h HeaderBinding) Name() string {
	return "header"
}

func (h HeaderBinding) Bind(header http.Header, obj interface{}) error {
	return mappingByPtr(obj, headerSource(header), "header")
}
