package render

import (
	"fmt"
	"github.com/xdatk/pisces/internal/bytesconv"
	"github.com/xdatk/pisces/internal/constant"
)

type Text struct {
	Format string
	Data   []interface{}
}

func (s Text) Render() ([]byte, error) {
	if len(s.Data) > 0 {
		str := fmt.Sprintf(s.Format, s.Data)
		return bytesconv.StringToBytes(str), nil
	}

	return bytesconv.StringToBytes(s.Format), nil
}

func (s Text) ContentType() string {
	return constant.MIMETextPlainCharsetUTF8
}
