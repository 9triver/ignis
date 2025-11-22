// Package store 提供了分布式对象存储的 Actor 实现
// 支持本地对象存储、远程对象获取、流式数据传输等功能
package store

import (
	pb "google.golang.org/protobuf/proto"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
)

// RequiresReplyMessage 是需要回复的消息接口
// 实现该接口的消息必须提供回复目标地址
type RequiresReplyMessage interface {
	pb.Message
	// GetReplyTo 返回回复目标的标识符
	GetReplyTo() string
}

// ForwardMessage 是需要转发的消息接口
// 实现该接口的消息必须提供转发目标地址
type ForwardMessage interface {
	pb.Message
	// GetTarget 返回转发目标的标识符
	GetTarget() string
}

// AddStore 是添加存储引用的消息
// 用于通知系统有新的 Store 可用
type AddStore struct {
	Ref *proto.StoreRef // Store 引用（包含 ID 和 PID）
}

// RemoveStore 是移除存储引用的消息
// 用于通知系统某个 Store 已不可用
type RemoveStore struct {
	ID string // Store 标识符
}

// SaveObject 是保存对象到 Store 的消息
// 当 Actor 生成新的返回对象时发送到 Store
type SaveObject struct {
	Value    object.Interface                         // 对象或流
	Callback func(ctx actor.Context, ref *proto.Flow) // 对象保存完成后的回调函数
}

// RequestObject 是从 Store 请求对象的消息
// 由本地 Actor 发送到 Store，请求获取指定的对象
type RequestObject struct {
	ReplyTo *actor.PID  // 回复目标的 PID
	Flow    *proto.Flow // Flow 引用（包含对象 ID 和来源 Store）
}

// ObjectResponse 是对象请求的响应消息
// Store 通过该消息将对象返回给请求者
type ObjectResponse struct {
	Value object.Interface // 对象值（如果成功）
	Error error            // 错误信息（如果失败）
}
