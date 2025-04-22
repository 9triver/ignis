package utils

type Function[I, O any] func(args I) (ret O, err error)

func (f Function[I, O]) Deps() []string {
	return fieldsOf[I]()
}

func (f Function[I, O]) Outputs() []string {
	return fieldsOf[O]()
}
