package binding

type UriBinding struct {
}

func (U UriBinding) Name() string {
	return "uri"
}

func (U UriBinding) Bind(params map[string][]string, obj interface{}) error {
	return mappingByPtr(obj, formSource(params), "tag")
}
