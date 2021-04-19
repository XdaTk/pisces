package binding

import (
	"encoding/json"
	"io"
)

type JsonBodyBinding struct {
	EnableDecoderUseNumber             bool
	EnableDecoderDisallowUnknownFields bool
}

func (j JsonBodyBinding) Name() string {
	return "json"
}

func (j JsonBodyBinding) Bind(body io.ReadCloser, obj interface{}) error {
	return j.decode(body, obj)
}

func (j JsonBodyBinding) decode(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if j.EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if j.EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(obj)
}
