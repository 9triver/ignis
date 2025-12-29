// Package functions 提供了函数调用的封装和适配功能
// 该文件实现了Go函数的包装器，支持将普通Go函数转换为平台可调用的函数接口
package functions

import (
	"reflect"
	"time"

	"github.com/9triver/ignis/actor/functions/gofunc"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type GoFunction struct {
	FuncDec
	impl     gofunc.WrappedFunc
	language proto.Language
}

func (f *GoFunction) Call(params map[string]object.Interface) (object.Interface, error) {
	invoke := make(map[string]any)
	for k, v := range params {
		var value any
		if s, ok := v.(*object.Stream); ok {
			value = s.ToChan()
		} else {
			vv, err := v.Value()
			if err != nil {
				return nil, errors.WrapWith(err, "call: failed fetching param %s", k)
			}
			value = vv
		}
		invoke[k] = value
	}

	// 添加日志以便调试
	// 注意：这里没有 context，所以不能使用 ctx.Logger()
	// 但可以通过其他方式记录日志，或者让调用者记录

	o, err := f.impl(invoke)
	if err != nil {
		return nil, errors.WrapWith(err, "call: execution failed")
	}

	t := reflect.TypeOf(o)
	var obj object.Interface
	if t.Kind() == reflect.Chan {
		obj = object.NewStream(o, f.language)
	} else {
		obj = object.NewLocal(o, f.language)
	}
	return obj, nil
}

func (f *GoFunction) TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error) {
	start := time.Now()
	obj, err := f.Call(params)
	return time.Since(start), obj, err
}

func (f *GoFunction) Language() proto.Language {
	return f.language
}

func ImplGo(
	name string,
	params []string,
	code string,
	language proto.Language,
) (*GoFunction, error) {
	impl, err := gofunc.Compile(name, code)
	if err != nil {
		return nil, err
	}

	return &GoFunction{
		FuncDec:  Declare(name, params),
		impl:     impl,
		language: language,
	}, nil
}

func NewGo[I, O any](
	name string,
	handler utils.Function[I, O],
	language proto.Language,
) *GoFunction {
	return &GoFunction{
		FuncDec:  DeclareTyped[I, O](name),
		impl:     gofunc.Wrap(handler),
		language: language,
	}
}
