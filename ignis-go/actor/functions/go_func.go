package functions

import (
	"reflect"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
	"github.com/asynkron/protoactor-go/actor"
)

type GoFunction[I, O any] struct {
	FuncDec
	impl     utils.Function[I, O]
	language proto.Language
}

var _ Function = (*GoFunction[any, any])(nil)

func (h *GoFunction[I, O]) Call(ctx actor.Context, sessionId string, params map[string]proto.Object) (proto.Object, error) {
	invoke := make(map[string]any)
	for k, v := range params {
		var value any
		if s, ok := v.ToStream(); ok {
			value = s.ToChan(ctx)
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
	var obj proto.Object
	if t.Kind() == reflect.Chan {
		obj = proto.NewLocalStream(o, h.language)
	} else {
		obj = proto.NewLocalObject(o, h.language)
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
