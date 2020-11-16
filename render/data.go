package render

type Data struct {
	Type string
	Data []byte
}

func (d Data) Render() ([]byte, error) {
	return d.Data, nil
}

func (d Data) ContentType() string {
	return d.Type
}
