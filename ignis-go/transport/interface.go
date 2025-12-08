// Package transport 提供了传输层的抽象接口和实现
// 支持多种传输协议（IPC、RPC、Websocket）和双向流式通信
//
// 主要组件:
//   - Protocol: 传输协议类型枚举
//   - Stream: 双向流式通信接口
//   - Manager: 传输管理器接口
//
// 该包为上层提供统一的传输抽象，隐藏底层通信细节
package transport

import (
	"context"

	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
)

// Protocol 表示传输协议类型
// 支持的协议包括：IPC（进程间通信）、RPC（远程过程调用）、Websocket（Web套接字）
type Protocol int

// 传输协议常量定义
const (
	IPC       Protocol = iota // IPC: 进程间通信（使用 ZeroMQ）
	RPC                       // RPC: 远程过程调用（使用 gRPC）
	Websocket                 // Websocket: Web 套接字通信
)

// Stream 是双向流式通信的泛型接口
// 类型参数:
//   - I: 输入消息类型（发送方向），必须是 protobuf Message
//   - O: 输出消息类型（接收方向），必须是 protobuf Message
//
// 该接口提供了基于 channel 的异步消息传递机制，支持：
//   - 通过 SendChan 发送消息到远端
//   - 通过 RecvChan 接收来自远端的消息
//   - 通过 Ready 信号等待连接就绪
type Stream[I, O pb.Message] interface {
	// Conn 返回连接标识符（如地址、ID 等）
	Conn() string

	// SendChan 返回发送消息的 channel
	// 向该 channel 发送消息将异步传输到远端
	SendChan() chan<- I

	// RecvChan 返回接收消息的 channel
	// 从该 channel 读取消息可获取远端发送的数据
	RecvChan() <-chan O

	// Ready 返回就绪信号 channel
	// 当连接建立并准备就绪时，该 channel 会被关闭
	Ready() <-chan struct{}

	// Protocol 返回当前使用的传输协议类型
	Protocol() Protocol
}

// Executor 是执行器的双向流接口
// 用于 Go 进程与 Python 执行器进程之间的通信
type Executor Stream[*executor.Message, *executor.Message]

// Controller 是控制器的双向流接口
// 用于平台控制器之间的通信
type Controller Stream[*controller.Message, *controller.Message]

// ComputeStream 是计算节点的双向流接口
// 用于集群计算节点之间的通信
type ComputeStream Stream[*cluster.Message, *cluster.Message]

// Manager 是传输管理器的基础接口
// 负责管理传输连接的生命周期和协议类型
//
// 实现者需要：
//   - 提供连接地址信息
//   - 实现运行逻辑（接受连接、处理消息等）
//   - 指定使用的传输协议类型
type Manager interface {
	// Addr 返回管理器的监听地址或连接地址
	Addr() string

	// Run 启动管理器并运行，直到 context 取消
	// 参数:
	//   - ctx: 上下文，用于控制生命周期
	// 返回值:
	//   - error: 运行过程中的错误
	Run(ctx context.Context) error

	// Type 返回管理器使用的传输协议类型
	Type() Protocol
}

// ExecutorManager 是执行器管理器接口
// 扩展了 Manager 接口，提供创建执行器流的能力
//
// 用于管理多个 Python 执行器的连接
type ExecutorManager interface {
	Manager

	// NewExecutor 创建或获取一个执行器流
	// 参数:
	//   - ctx: 上下文，用于控制执行器生命周期
	//   - conn: 连接标识符（如虚拟环境名称）
	// 返回值:
	//   - Executor: 执行器双向流接口
	NewExecutor(ctx context.Context, conn string) Executor
}

// ControllerManager 是控制器管理器接口
// 扩展了 Manager 接口，提供获取控制器流的能力
//
// 用于管理平台控制器之间的连接
type ControllerManager interface {
	Manager

	// NextController 获取下一个可用的控制器流
	// 返回值:
	//   - Controller: 控制器双向流接口
	NextController() Controller
}
