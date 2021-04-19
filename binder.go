package pisces

import (
	"fmt"
	"net/http"

	"github.com/xdatk/pisces/binding"
	"github.com/xdatk/pisces/internal/constant"
)

type Binder struct {
	Param         binding.Param
	Header        binding.Header
	Query         binding.Form
	Form          binding.Form
	PostForm      binding.Form
	MultipartFrom binding.MultipartFrom
	Body          map[string]binding.Body
}

func (b Binder) Bind(c *Context, obj interface{}) error {
	if c.Request.Method == http.MethodGet {
		return b.Form.Bind(c.GetQuerys(), obj)
	}

	contentType := c.ContentType()
	switch contentType {
	case constant.MIMEApplicationForm, constant.MIMEMultipartForm:
		form, err := c.GetForms()
		if err != nil {
			return err
		}
		return b.Form.Bind(form, obj)
	default:
		binder, ok := b.Body[contentType]
		if !ok {
			return fmt.Errorf("not support content type")
		}
		return binder.Bind(c.Body(), obj)
	}
}
