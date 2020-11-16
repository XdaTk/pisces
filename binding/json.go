package binding

import (
	"encoding/json"
	"io"
)

type JSON struct {
	EnableDecoderUseNumber             bool
	EnableDecoderDisallowUnknownFields bool
}

func (j JSON) Name() string {
	return "json"
}

func (j JSON) Bind(body io.ReadCloser, obj interface{}) error {
	return j.decode(body, obj)
}

func (j JSON) decode(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if j.EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if j.EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(obj)
}
