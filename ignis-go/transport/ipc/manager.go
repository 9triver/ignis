// Package ipc 提供基于 ZeroMQ 的进程间通信（IPC）传输实现
package ipc

import (
	"context"
	_ "embed"
	"sync"

	pb "google.golang.org/protobuf/proto"
	"gopkg.in/zeromq/goczmq.v4"

	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/transport"
)

// ConnectionManager 是基于 ZeroMQ Router 模式的 IPC 连接管理器
// 负责管理多个 Python 执行器的连接和消息路由
//
// 工作原理:
//   - 使用 ZeroMQ Router 套接字接收来自多个客户端的消息
//   - 为每个连接维护一个 ExecutorImpl 实例
//   - 根据消息中的连接标识符路由消息到对应的执行器
type ConnectionManager struct {
	addr      string                             // 监听地址
	executors map[string]*transport.ExecutorImpl // 执行器映射表，key 为连接标识符
	mu        sync.RWMutex                       // 保护 executors 的读写锁
}

// Addr 返回管理器的监听地址
func (cm *ConnectionManager) Addr() string {
	return cm.addr
}

// onReceive 处理从 ZeroMQ Router 接收到的消息
// 参数:
//   - router: ZeroMQ Router channeler，用于发送响应
//   - frame: 消息帧（用于路由回复）
//   - msg: 解析后的执行器消息
//
// 消息类型处理:
//   - Ready: 设置执行器的发送函数，建立双向通信
//   - Return/StreamChunk: 将消息传递给对应的执行器
func (cm *ConnectionManager) onReceive(router *goczmq.Channeler, frame []byte, msg *executor.Message) {
	conn := msg.Conn

	cm.mu.RLock()
	e, ok := cm.executors[conn]
	cm.mu.RUnlock()

	if !ok {
		// TODO: 记录日志，未知的连接
		return
	}

	switch msg.Command.(type) {
	case *executor.Message_Ready:
		// 设置发送函数，通过 Router 发送消息回客户端
		e.SetSender(func(msg *executor.Message) error {
			data, err := pb.Marshal(msg)
			if err != nil {
				return err
			}
			router.SendChan <- [][]byte{frame, data}
			return nil
		})
	case *executor.Message_Return, *executor.Message_StreamChunk:
		e.Produce(msg)
	}
}

// Start 在后台启动连接管理器
// 参数:
//   - ctx: 上下文，用于控制生命周期
//
// 返回值:
//   - error: 启动错误（注意：此方法有 bug，不会返回实际的运行错误）
//
// 已知问题: goroutine 中的 err 不会被返回，建议直接使用 Run 方法
// Deprecated: 使用 Run 方法代替
func (cm *ConnectionManager) Start(ctx context.Context) error {
	var err error
	go func() {
		err = cm.Run(ctx)
	}()
	return err
}

// Run 启动连接管理器并阻塞运行
// 参数:
//   - ctx: 上下文，用于控制生命周期
//
// 返回值:
//   - error: 运行过程中的错误
//
// 执行流程:
//  1. 创建 ZeroMQ Router 套接字
//  2. 持续接收并处理来自客户端的消息
//  3. 当 context 取消时退出
//  4. 通过 defer 自动关闭所有执行器并清理 map，防止内存泄漏
//
// 注意: 该方法会阻塞，应在独立的 goroutine 中调用
func (cm *ConnectionManager) Run(ctx context.Context) error {
	router := goczmq.NewRouterChanneler(cm.addr)
	defer router.Destroy()

	defer func() {
		cm.mu.Lock()
		defer cm.mu.Unlock()

		// 关闭所有执行器
		for _, e := range cm.executors {
			_ = e.Close()
		}
		// 清空 map，释放内存
		cm.executors = make(map[string]*transport.ExecutorImpl)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-router.RecvChan:
			if len(msg) < 2 {
				// 消息格式不正确，跳过
				continue
			}
			frame, data := msg[0], msg[1]
			cmd := &executor.Message{}
			if err := pb.Unmarshal(data, cmd); err != nil {
				// 解析失败，跳过
				// TODO: 记录日志
				continue
			}
			cm.onReceive(router, frame, cmd)
		}
	}
}

// NewExecutor 创建或获取一个执行器实例
// 参数:
//   - ctx: 上下文，用于控制执行器生命周期
//   - conn: 连接标识符（通常是虚拟环境名称）
//
// 返回值:
//   - transport.Executor: 执行器实例
//
// 如果执行器已存在则直接返回，否则创建新的执行器并启动
// 该方法是线程安全的
//
// 注意: 所有执行器会在 Run 方法返回时通过 defer 统一清理
func (cm *ConnectionManager) NewExecutor(ctx context.Context, conn string) transport.Executor {
	// 先尝试读锁获取
	cm.mu.RLock()
	e, ok := cm.executors[conn]
	cm.mu.RUnlock()

	if ok {
		return e
	}

	// 需要创建新执行器，使用写锁
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 双重检查，防止并发创建
	if e, ok := cm.executors[conn]; ok {
		return e
	}

	e = transport.NewExecutorImpl(conn, transport.IPC)
	go e.Run(ctx)
	cm.executors[conn] = e
	return e
}

// Type 返回传输协议类型
func (cm *ConnectionManager) Type() transport.Protocol {
	return transport.IPC
}

// NewManager 创建一个新的 IPC 连接管理器
// 参数:
//   - addr: ZeroMQ 监听地址（例如：tcp://127.0.0.1:5555）
//
// 返回值:
//   - *ConnectionManager: 管理器实例
func NewManager(addr string) *ConnectionManager {
	return &ConnectionManager{
		addr:      addr,
		executors: make(map[string]*transport.ExecutorImpl),
	}
}

// PythonExecutorTemplate 是嵌入的 Python 执行器模板脚本
// 该脚本会被写入到 Python 虚拟环境中，作为执行器的启动脚本
var (
	//go:embed executor_template.py
	PythonExecutorTemplate string
)
