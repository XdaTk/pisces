package binding

import (
	"mime/multipart"
)

type MultipartFormBinding struct {
}

func (m MultipartFormBinding) Name() string {
	return "multipart/form-data"
}

func (m MultipartFormBinding) Bind(form *multipart.Form, obj interface{}) error {
	return mappingByPtr(obj, (*multipartFormSource)(form), "form")
}
