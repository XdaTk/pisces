package binding

import (
	"net/url"
)

type FormBinding struct {
}

func (f FormBinding) Name() string {
	return "form"
}

func (f FormBinding) Bind(values url.Values, obj interface{}) error {
	return mappingByPtr(obj, formSource(values), "form")
}
