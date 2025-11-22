// Package transport 提供了传输层的抽象接口和实现
package transport

import (
	"context"
	"sync"
	"time"

	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils/errors"
)

const (
	// connectionTimeout 是连接就绪的最大等待时间
	connectionTimeout = 30 * time.Second
)

// StreamImpl 是 Stream 接口的泛型实现
// 类型参数:
//   - I: 输入消息类型（发送方向），必须是 protobuf Message
//   - O: 输出消息类型（接收方向），必须是 protobuf Message
//
// 工作原理:
//   - 使用 channel 进行异步消息传递
//   - sender 函数负责实际的消息发送（由具体协议实现注入）
//   - ready channel 用于同步连接就绪状态
//   - send/recv channel 提供缓冲的消息队列
type StreamImpl[I, O pb.Message] struct {
	conn       string            // 连接标识符
	protocol   Protocol          // 传输协议类型
	sender     func(msg I) error // 消息发送函数（由外部设置）
	ready      chan struct{}     // 就绪信号 channel
	send       chan I            // 发送消息队列
	recv       chan O            // 接收消息队列
	closeOnce  sync.Once         // 确保只关闭一次
	senderOnce sync.Once         // 确保只设置一次 sender
}

// Conn 返回连接标识符
func (s *StreamImpl[I, O]) Conn() string {
	return s.conn
}

// SendChan 返回发送消息的 channel
// 向该 channel 发送消息将被异步传输到远端
func (s *StreamImpl[I, O]) SendChan() chan<- I {
	return s.send
}

// RecvChan 返回接收消息的 channel
// 从该 channel 读取消息可获取远端发送的数据
func (s *StreamImpl[I, O]) RecvChan() <-chan O {
	return s.recv
}

// Ready 返回就绪信号 channel
// 当连接建立并设置了 sender 后，该 channel 会被关闭
func (s *StreamImpl[I, O]) Ready() <-chan struct{} {
	return s.ready
}

// Protocol 返回当前使用的传输协议类型
func (s *StreamImpl[I, O]) Protocol() Protocol {
	return s.protocol
}

// SetSender 设置消息发送函数
// 参数:
//   - sender: 实际的消息发送函数，由具体的传输协议实现提供
//
// 该方法是线程安全的，只能调用一次，重复调用会被忽略
// 设置后会关闭 ready channel 通知连接就绪
func (s *StreamImpl[I, O]) SetSender(sender func(msg I) error) {
	s.senderOnce.Do(func() {
		s.sender = sender
		close(s.ready)
	})
}

// Produce 将接收到的消息放入接收队列
// 参数:
//   - msg: 接收到的消息
//
// 该方法由底层传输协议实现调用，将远端消息传递给上层
func (s *StreamImpl[I, O]) Produce(msg O) {
	s.recv <- msg
}

// Run 启动流的消息发送循环
// 参数:
//   - ctx: 上下文，用于控制生命周期
//
// 返回值:
//   - error: 运行过程中的错误
//
// 执行流程:
//  1. 等待连接就绪（最多 30 秒超时）
//  2. 持续从 send channel 读取消息并通过 sender 发送
//  3. 当 context 取消时退出
//
// 注意: 该方法会阻塞运行，通常在 goroutine 中调用
func (s *StreamImpl[I, O]) Run(ctx context.Context) error {
	select {
	case <-time.After(connectionTimeout):
		return errors.New("connection timeout")
	case <-s.Ready():
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-s.send:
			if err := s.sender(msg); err != nil {
				return err
			}
		}
	}
}

// Close 关闭流并释放资源
// 返回值:
//   - error: 关闭过程中的错误
//
// 该方法是线程安全的，可以多次调用但只会关闭一次
func (s *StreamImpl[I, O]) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.recv != nil {
			close(s.recv)
		}
	})
	return err
}

// NewStreamImpl 创建一个新的流实例
// 类型参数:
//   - I: 输入消息类型（发送方向）
//   - O: 输出消息类型（接收方向）
//
// 参数:
//   - conn: 连接标识符
//   - protocol: 传输协议类型
//
// 返回值:
//   - *StreamImpl[I, O]: 流实例
//
// 注意: 创建后需要调用 SetSender 设置发送函数，然后调用 Run 启动
func NewStreamImpl[I, O pb.Message](conn string, protocol Protocol) *StreamImpl[I, O] {
	bufferSize := configs.ChannelBufferSize
	return &StreamImpl[I, O]{
		conn:     conn,
		protocol: protocol,
		ready:    make(chan struct{}),
		send:     make(chan I, bufferSize),
		recv:     make(chan O, bufferSize),
	}
}

// ExecutorImpl 是执行器流的具体实现
// 用于 Go 进程与 Python 执行器进程之间的双向通信
type ExecutorImpl = StreamImpl[*executor.Message, *executor.Message]

// ControllerImpl 是控制器流的具体实现
// 用于平台控制器之间的双向通信
type ControllerImpl = StreamImpl[*controller.Message, *controller.Message]

// ComputeStreamImpl 是计算流的具体实现
// 用于集群计算节点之间的双向通信
type ComputeStreamImpl = StreamImpl[*cluster.Message, *cluster.Message]

// NewExecutorImpl 创建一个新的执行器流实例
// 参数:
//   - conn: 连接标识符（如虚拟环境名称）
//   - protocol: 传输协议类型
//
// 返回值:
//   - *ExecutorImpl: 执行器流实例
func NewExecutorImpl(conn string, protocol Protocol) *ExecutorImpl {
	return NewStreamImpl[*executor.Message, *executor.Message](conn, protocol)
}

// NewControllerImpl 创建一个新的控制器流实例
// 参数:
//   - conn: 连接标识符
//   - protocol: 传输协议类型
//
// 返回值:
//   - *ControllerImpl: 控制器流实例
func NewControllerImpl(conn string, protocol Protocol) *ControllerImpl {
	return NewStreamImpl[*controller.Message, *controller.Message](conn, protocol)
}

// NewComputeStreamImpl 创建一个新的计算流实例
// 参数:
//   - conn: 连接标识符
//   - protocol: 传输协议类型
//
// 返回值:
//   - *ComputeStreamImpl: 计算流实例
func NewComputeStreamImpl(conn string, protocol Protocol) *ComputeStreamImpl {
	return NewStreamImpl[*cluster.Message, *cluster.Message](conn, protocol)
}
