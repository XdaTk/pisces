package binding

import (
	"mime/multipart"
)

type MultipartForm struct {
}

func (m MultipartForm) Name() string {
	return "multipart/form-data"
}

func (m MultipartForm) Bind(form *multipart.Form, obj interface{}) error {
	return mappingByPtr(obj, (*multipartFormSource)(form), "form")
}
