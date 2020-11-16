package binding

type URI struct {
}

func (U URI) Name() string {
	return "uri"
}

func (U URI) Bind(params map[string][]string, obj interface{}) error {
	return mappingByPtr(obj,formSource(params),"tag")
}
