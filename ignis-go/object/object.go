// Package object 提供了对象的统一接口和实现
// 支持本地对象、远程对象和流式对象的序列化、传输和管理
package object

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// Interface 是对象的统一接口
// 封装了 Local、Stream 和 Remote（proto.EncodedObject）三种对象类型
//
// 所有实现都支持序列化和反序列化，用于跨进程、跨语言的数据传输
//
// 注意:
//   - 对象的编码/解码可能开销较大
//   - 该接口主要用于 Actor 函数调用时的参数和返回值传递
//
// 支持的语言:
//   - Go: 使用 gob 编码
//   - Python: 使用 pickle 编码（字节数组）
//   - JSON: 使用 JSON 编码
type Interface interface {
	// GetID 返回对象的唯一标识符
	GetID() string

	// GetLanguage 返回对象所属的语言类型
	GetLanguage() Language

	// Encode 将对象编码为可传输的 Remote 格式
	// 返回值:
	//   - *Remote: 编码后的远程对象
	//   - error: 编码错误
	Encode() (*Remote, error)

	// Value 返回对象的原始值
	// 返回值:
	//   - any: 对象值
	//   - error: 获取错误
	Value() (any, error)
}

// 编译时检查类型是否实现了 Interface 接口
var (
	_ Interface = (*Local)(nil)
	_ Interface = (*Remote)(nil)
	_ Interface = (*Stream)(nil)
)

// 类型别名，简化使用
type (
	// Remote 是远程对象的别名，实际为 proto.EncodedObject
	// 表示已编码的、可在网络上传输的对象
	Remote = proto.EncodedObject

	// Language 是编程语言类型的别名
	Language = proto.Language
)

// 支持的编程语言常量
const (
	LangUnknown = proto.Language_LANG_UNKNOWN // 未知语言
	LangJson    = proto.Language_LANG_JSON    // JSON 格式
	LangGo      = proto.Language_LANG_GO      // Go 语言（使用 gob 编码）
	LangPython  = proto.Language_LANG_PYTHON  // Python 语言（使用 pickle 编码）
)

// Local 表示本地对象
// 存储在当前进程内存中的对象，支持编码后传输到其他进程
type Local struct {
	id       string   // 对象唯一标识符
	value    any      // 对象值
	language Language // 对象所属的语言类型
}

// GetID 返回对象的唯一标识符
func (obj *Local) GetID() string {
	return obj.id
}

// Value 返回对象的原始值
func (obj *Local) Value() (any, error) {
	return obj.value, nil
}

// Encode 将本地对象编码为可传输的 Remote 格式
// 返回值:
//   - *Remote: 编码后的远程对象
//   - error: 编码错误
//
// 根据语言类型选择不同的编码方式:
//   - LangJson: 使用 JSON 编码
//   - LangPython: 使用 pickle 编码（要求 value 为 []byte）
//   - LangGo: 使用 gob 编码
func (obj *Local) Encode() (*Remote, error) {
	o := &Remote{
		ID:       obj.id,
		Language: obj.language,
	}

	switch obj.language {
	case LangJson:
		data, err := json.Marshal(obj.value)
		if err != nil {
			return nil, errors.WrapWith(err, "encoder: json failed")
		}
		o.Data = data
	case LangPython:
		if data, ok := obj.value.([]byte); !ok {
			return nil, errors.New("encoder: python object must be pickled bytes")
		} else {
			o.Data = data
		}
	case LangGo:
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(obj.value); err != nil {
			return nil, errors.WrapWith(err, "encoder: gob failed")
		}
		o.Data = buf.Bytes()
	default:
		return nil, errors.New("encoder: unsupported language")
	}
	return o, nil
}

// GetLanguage 返回对象所属的语言类型
func (obj *Local) GetLanguage() Language {
	return obj.language
}

// NewLocal 创建一个新的本地对象，自动生成唯一 ID
// 参数:
//   - value: 对象值
//   - language: 语言类型
//
// 返回值:
//   - *Local: 本地对象实例
//
// ID 格式: "obj." + 随机字符串
func NewLocal(value any, language Language) *Local {
	return LocalWithID(utils.GenIDWith("obj."), value, language)
}

// LocalWithID 创建一个带指定 ID 的本地对象
// 参数:
//   - id: 对象唯一标识符
//   - value: 对象值
//   - language: 语言类型
//
// 返回值:
//   - *Local: 本地对象实例
func LocalWithID(id string, value any, language Language) *Local {
	return &Local{
		id:       id,
		value:    value,
		language: language,
	}
}
