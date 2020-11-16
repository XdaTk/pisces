package pisces

type Validator interface {
	Validate(interface{}) error
}
