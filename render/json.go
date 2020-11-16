package render

import (
	"encoding/json"
	"github.com/xdatk/pisces/internal/constant"
)

type JSON struct {
	Data interface{}
}

func (j JSON) Render() ([]byte, error) {
	return json.Marshal(j.Data)
}

func (j JSON) ContentType() string {
	return constant.MIMEApplicationJSONCharsetUTF8
}
