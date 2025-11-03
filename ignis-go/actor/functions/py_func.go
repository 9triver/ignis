// Package functions 提供了函数调用的封装和适配功能
// 该文件实现了 Python 函数的包装器，支持在虚拟环境中执行 Python 函数
package functions

import (
	"strings"
	"time"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils/errors"
)

// PyFunction 是 Python 函数的调用包装器
// 用于封装和执行 Python 虚拟环境中的函数
//
// 工作原理:
//   - Python 函数以序列化（pickled）形式存储在虚拟环境中
//   - 调用时通过虚拟环境的执行器远程执行 Python 代码
//   - 支持对象方法调用（obj.method 格式）
//   - 自动处理参数的序列化和反序列化
type PyFunction struct {
	FuncDec
	venv     *VirtualEnv    // Python 虚拟环境引用
	language proto.Language // 函数所属的语言类型
}

// NewPy 创建一个新的 Python 函数包装器实例
// 参数:
//   - manager: 虚拟环境管理器，用于管理 Python 虚拟环境
//   - name: 函数名称，支持 "obj.method" 格式调用对象方法
//   - params: 函数参数名列表
//   - venv: 虚拟环境名称
//   - packages: 需要安装的 Python 包依赖列表
//   - pickledObj: 序列化（pickled）的 Python 函数对象
//   - language: 函数所属的语言类型
//
// 返回值:
//   - *PyFunction: 创建的 Python 函数包装器实例
//   - error: 创建过程中的错误
//
// 该函数会自动生成函数声明并调用 ImplPy 完成实际创建
func NewPy(
	manager *VenvManager,
	name string,
	params []string,
	venv string,
	packages []string,
	pickledObj []byte,
	language proto.Language,
) (*PyFunction, error) {
	dec := Declare(name, params)
	return ImplPy(manager, dec, venv, packages, pickledObj, language)
}

// ImplPy 使用指定的函数声明创建 Python 函数包装器实例
// 参数:
//   - manager: 虚拟环境管理器
//   - dec: 函数声明（FuncDec），包含函数的元数据信息
//   - venv: 虚拟环境名称
//   - packages: 需要安装的 Python 包依赖列表
//   - pickledObj: 序列化（pickled）的 Python 函数对象
//   - language: 函数所属的语言类型
//
// 返回值:
//   - *PyFunction: 创建的 Python 函数包装器实例
//   - error: 创建过程中的错误
//
// 执行流程:
//  1. 获取或创建指定的虚拟环境（自动安装依赖包）
//  2. 创建 AddHandler 请求，将序列化的 Python 函数对象发送到虚拟环境
//  3. 返回封装好的 PyFunction 实例
func ImplPy(
	manager *VenvManager,
	dec FuncDec,
	venv string,
	packages []string,
	pickledObj []byte,
	language proto.Language,
) (*PyFunction, error) {
	name := dec.Name()
	env, err := manager.GetVenv(venv, packages...)
	if err != nil {
		return nil, errors.WrapWith(err, "%s: error creating impl", name)
	}

	addHandler := executor.NewAddHandler(venv, name, pickledObj, language, nil)
	env.Send(addHandler)

	return &PyFunction{
		FuncDec:  dec,
		venv:     env,
		language: language,
	}, nil
}

// Call 执行封装的 Python 函数
// 参数:
//   - params: 输入参数 map，key 为参数名，value 为参数值对象
//
// 返回值:
//   - object.Interface: 函数执行结果对象
//   - error: 执行过程中的错误
//
// 执行流程:
//  1. 解析函数名：
//     - 如果是 "obj.method" 格式，分别提取对象名和方法名
//     - 如果是普通函数名，将其作为对象名，方法名为空
//  2. 通过虚拟环境的执行器调用 Python 函数
//  3. 等待并返回执行结果
//
// 注意: 参数和返回值会自动进行序列化/反序列化转换
func (f *PyFunction) Call(params map[string]object.Interface) (object.Interface, error) {
	segs := strings.Split(f.Name(), ".")
	var obj, method string
	if len(segs) >= 2 {
		obj, method = segs[0], segs[1]
	} else {
		obj, method = f.Name(), ""
	}
	result, err := f.venv.Execute(obj, method, params).Result()
	if err != nil {
		return nil, errors.WrapWith(err, "%s: execution failed", f.name)
	}
	return result, nil
}

// TimedCall 执行封装的 Python 函数并统计执行时间
// 参数:
//   - params: 输入参数 map，key 为参数名，value 为参数值对象
//
// 返回值:
//   - time.Duration: 函数执行耗时
//   - object.Interface: 函数执行结果对象
//   - error: 执行过程中的错误
func (f *PyFunction) TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error) {
	start := time.Now()
	obj, err := f.Call(params)
	return time.Since(start), obj, err
}

// Language 返回函数所属的语言类型
func (f *PyFunction) Language() proto.Language {
	return f.language
}
