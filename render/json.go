package render

import (
	"encoding/json"
	"github.com/xdatk/pisces/internal/constant"
)

type Json struct {
	Data interface{}
}

func (j Json) Render() ([]byte, error) {
	return json.Marshal(j.Data)
}

func (j Json) ContentType() string {
	return constant.MIMEApplicationJSONCharsetUTF8
}
