package render

type Render interface {
	Render() ([]byte, error)
	ContentType() string
}
