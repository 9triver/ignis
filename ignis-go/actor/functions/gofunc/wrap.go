package gofunc

import "github.com/9triver/ignis/utils"

type WrappedFunc = func(inputs map[string]any) (any, error)

func Wrap[I, O any](f utils.Function[I, O]) WrappedFunc {
	return func(inputs map[string]any) (any, error) {
		input, err := utils.MapToStruct[I](inputs)
		if err != nil {
			return nil, err
		}

		return f(input)
	}
}
