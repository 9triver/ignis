// Package functions 提供了函数调用的封装和适配功能
// 该文件实现了Go函数的包装器，支持将普通Go函数转换为平台可调用的函数接口
package functions

import (
	"reflect"
	"time"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// GoFunction 是一个Go函数的调用包装器，用于封装和执行Go函数
// 类型参数:
//   - I: 输入参数类型，应为struct类型，用于接收函数的输入参数
//   - O: 输出结果类型，可以是任意类型（struct、chan、基本类型等），表示函数的返回值
//
// 该结构体将普通的Go函数适配为平台可调用的函数接口，
// 支持参数的自动映射、类型转换以及执行时间统计等功能
// 当O为channel类型时，返回结果将被封装为Stream对象；否则封装为Local对象
type GoFunction[I, O any] struct {
	FuncDec
	impl     utils.Function[I, O] // 实际执行的Go函数实现
	language proto.Language       // 函数所属的语言类型
}

// Call 执行封装的Go函数
// 参数:
//   - params: 输入参数map，key为参数名，value为参数值对象
//
// 返回值:
//   - object.Interface: 函数执行结果对象
//   - error: 执行过程中的错误
//
// 执行流程:
//  1. 将输入参数从objects.Interface转换为原始类型
//  2. 将参数map映射为类型I的struct
//  3. 调用实际的Go函数实现
//  4. 将返回结果封装为objects.Interface对象（Local或Stream）
func (h *GoFunction[I, O]) Call(params map[string]object.Interface) (object.Interface, error) {
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

	input, err := utils.MapToStruct[I](invoke)
	if err != nil {
		return nil, errors.WrapWith(err, "call: failed decoding params")
	}

	o, err := h.impl(input)
	if err != nil {
		return nil, errors.WrapWith(err, "call: execution failed")
	}

	t := reflect.TypeFor[O]()
	var obj object.Interface
	if t.Kind() == reflect.Chan {
		obj = object.NewStream(o, h.language)
	} else {
		obj = object.NewLocal(o, h.language)
	}
	return obj, nil
}

// TimedCall 执行封装的Go函数并统计执行时间
// 参数:
//   - params: 输入参数map，key为参数名，value为参数值对象
//
// 返回值:
//   - time.Duration: 函数执行耗时
//   - object.Interface: 函数执行结果对象
//   - error: 执行过程中的错误
func (h *GoFunction[I, O]) TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error) {
	start := time.Now()
	obj, err := h.Call(params)
	return time.Since(start), obj, err
}

// Language 返回函数所属的语言类型
func (h *GoFunction[I, O]) Language() proto.Language {
	return h.language
}

// ImplGo 创建一个带有指定函数声明的GoFunction实例
// 类型参数:
//   - I: 输入参数struct类型
//   - O: 输出结果类型，可以是任意类型
//
// 参数:
//   - def: 函数声明（FuncDec），包含函数的元数据信息
//   - handler: 实际的Go函数实现
//   - language: 函数所属的语言类型
//
// 返回值:
//   - *GoFunction[I, O]: 创建的GoFunction实例
func ImplGo[I, O any](
	def FuncDec,
	handler utils.Function[I, O],
	language proto.Language,
) *GoFunction[I, O] {
	return &GoFunction[I, O]{FuncDec: def, impl: handler, language: language}
}

// NewGo 创建一个新的GoFunction实例，自动生成函数声明
// 类型参数:
//   - I: 输入参数struct类型
//   - O: 输出结果类型，可以是任意类型
//
// 参数:
//   - name: 函数名称
//   - handler: 实际的Go函数实现
//   - language: 函数所属的语言类型
//
// 返回值:
//   - *GoFunction[I, O]: 创建的GoFunction实例
//
// 该函数会根据类型参数I和O自动生成函数声明（DeclareTyped）
func NewGo[I, O any](
	name string,
	handler utils.Function[I, O],
	language proto.Language,
) *GoFunction[I, O] {
	return &GoFunction[I, O]{FuncDec: DeclareTyped[I, O](name), impl: handler, language: language}
}
