package functions

import (
	"reflect"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type GoFunction[I, O any] struct {
	FuncDec
	impl     utils.Function[I, O]
	language proto.Language
}

var _ Function = (*GoFunction[any, any])(nil)

func (h *GoFunction[I, O]) Call(params map[string]messages.Object) (messages.Object, error) {
	invoke := make(map[string]any)
	for k, v := range params {
		var value any
		if s, ok := v.(*messages.LocalStream); ok {
			value = s.ToChan()
		} else {
			vv, err := v.GetValue()
			if err != nil {
				return nil, errors.WrapWith(err, "call: failed fetching param %s", k)
			}
			value = vv
		}
		invoke[k] = value
	}

	input, err := utils.MapToStruct[I](invoke)
	if err != nil {
		return nil, errors.WrapWith(err, "call: failed decoding params")
	}

	o, err := h.impl(input)
	if err != nil {
		return nil, errors.WrapWith(err, "call: execution failed")
	}

	t := reflect.TypeFor[O]()
	var obj messages.Object
	if t.Kind() == reflect.Chan {
		obj = messages.NewLocalStream(o, h.language)
	} else {
		obj = messages.NewLocalObject(o, h.language)
	}
	return obj, nil
}

func (h *GoFunction[I, O]) Language() proto.Language {
	return h.language
}

func ImplGo[I, O any](
	def FuncDec,
	handler utils.Function[I, O],
	language proto.Language,
) *GoFunction[I, O] {
	return &GoFunction[I, O]{FuncDec: def, impl: handler, language: language}
}

func NewGo[I, O any](
	name string,
	handler utils.Function[I, O],
	language proto.Language,
) *GoFunction[I, O] {
	return &GoFunction[I, O]{FuncDec: DeclareTyped[I, O](name), impl: handler, language: language}
}
