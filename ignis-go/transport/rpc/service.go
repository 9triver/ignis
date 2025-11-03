// Package rpc 提供基于 gRPC 的服务实现
package rpc

import (
	"context"
	"io"
	"sync"

	"google.golang.org/grpc"

	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/utils"
	"github.com/sirupsen/logrus"
)

// controllerService 实现了 gRPC 控制器服务
// 管理多个控制器连接，支持双向流式通信
type controllerService struct {
	controller.UnimplementedServiceServer
	controllers map[string]*transport.ControllerImpl // 控制器映射表
	next        chan *transport.ControllerImpl       // 新控制器通知 channel
	mu          sync.RWMutex                         // 保护 controllers 的读写锁
}

// Session 处理控制器的双向流式会话
// 参数:
//   - stream: gRPC 双向流
//
// 返回值:
//   - error: 会话过程中的错误
//
// 执行流程:
//  1. 生成唯一的连接 ID
//  2. 创建控制器流并设置发送函数
//  3. 将新连接发送到 next channel 通知调用者
//  4. 启动流的发送循环
//  5. 持续接收并转发消息到控制器流
//  6. 会话结束时自动清理（defer）
func (cs *controllerService) Session(stream grpc.BidiStreamingServer[controller.Message, controller.Message]) error {
	conn := utils.GenID()
	c := cs.newConn(conn)
	c.SetSender(stream.Send)

	// 非阻塞发送到 next channel，避免永久阻塞
	select {
	case cs.next <- c:
		// 成功通知有新控制器连接
	case <-stream.Context().Done():
		// context 已取消，直接返回
		c.Close()
		cs.mu.Lock()
		delete(cs.controllers, conn)
		cs.mu.Unlock()
		return stream.Context().Err()
	}

	defer func() {
		c.Close()
		// 从 map 中清理该连接，防止内存泄漏
		cs.mu.Lock()
		delete(cs.controllers, conn)
		cs.mu.Unlock()
	}()

	go c.Run(stream.Context())

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.Produce(msg)
	}
}

// newConn 创建或获取一个控制器流实例
// 参数:
//   - conn: 连接标识符
//
// 返回值:
//   - *transport.ControllerImpl: 控制器流实例
func (cs *controllerService) newConn(conn string) *transport.ControllerImpl {
	cs.mu.RLock()
	c, ok := cs.controllers[conn]
	cs.mu.RUnlock()

	if ok {
		return c
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 双重检查
	if c, ok := cs.controllers[conn]; ok {
		return c
	}

	c = transport.NewControllerImpl(conn, transport.RPC)
	cs.controllers[conn] = c
	return c
}

// nextConn 阻塞等待并返回下一个可用的控制器连接
// 返回值:
//   - *transport.ControllerImpl: 控制器流实例
func (cs *controllerService) nextConn() *transport.ControllerImpl {
	return <-cs.next
}

// close 关闭控制器服务并清理所有连接
func (cs *controllerService) close() {
	close(cs.next)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	for _, c := range cs.controllers {
		_ = c.Close()
	}
	// 清空 map，释放内存
	cs.controllers = make(map[string]*transport.ControllerImpl)
}

// executorService 实现了 gRPC 执行器服务
// 管理多个 Python 执行器的连接
type executorService struct {
	executor.UnimplementedServiceServer
	executors map[string]*transport.ExecutorImpl // 执行器映射表
	mu        sync.RWMutex                       // 保护 executors 的读写锁
}

// Session 处理执行器的双向流式会话
// 参数:
//   - stream: gRPC 双向流
//
// 返回值:
//   - error: 会话过程中的错误
//
// 执行流程:
//   - 持续接收来自客户端的消息
//   - 根据消息类型分发到对应的处理器
//   - 当接收到 EOF 或错误时退出
func (es *executorService) Session(stream grpc.BidiStreamingServer[executor.Message, executor.Message]) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		es.onReceive(stream, msg)
	}
}

// onReceive 处理接收到的执行器消息
// 参数:
//   - stream: gRPC 双向流
//   - msg: 执行器消息
//
// 消息类型处理:
//   - Ready: 设置执行器的发送函数
//   - 其他: 将消息转发到对应的执行器
func (es *executorService) onReceive(stream grpc.BidiStreamingServer[executor.Message, executor.Message], msg *executor.Message) {
	es.mu.RLock()
	c, ok := es.executors[msg.Conn]
	es.mu.RUnlock()

	if !ok {
		// TODO: 记录日志，未知的连接
		return
	}

	switch msg.Command.(type) {
	case *executor.Message_Ready:
		c.SetSender(stream.Send)
	default:
		c.Produce(msg)
	}
}

// newConn 创建或获取一个执行器流实例
// 参数:
//   - ctx: 上下文，用于控制执行器生命周期
//   - conn: 连接标识符
//
// 返回值:
//   - transport.Executor: 执行器实例
//
// 注意: 所有执行器会在 close 方法中统一清理
func (es *executorService) newConn(ctx context.Context, conn string) transport.Executor {
	es.mu.RLock()
	c, ok := es.executors[conn]
	es.mu.RUnlock()

	if ok {
		return c
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// 双重检查
	if c, ok := es.executors[conn]; ok {
		return c
	}

	c = transport.NewExecutorImpl(conn, transport.RPC)
	go c.Run(ctx)
	es.executors[conn] = c
	return c
}

// close 关闭执行器服务并清理所有连接
func (es *executorService) close() {
	es.mu.Lock()
	defer es.mu.Unlock()

	for _, c := range es.executors {
		_ = c.Close()
	}
	// 清空 map，释放内存
	es.executors = make(map[string]*transport.ExecutorImpl)
}

// computeService 实现了 gRPC 计算服务
// 管理多个集群计算节点的连接
type computeService struct {
	cluster.UnimplementedServiceServer
	computers map[string]*transport.ComputeStreamImpl // 计算节点映射表
	mu        sync.RWMutex                            // 保护 computers 的读写锁
}

// Session 处理计算节点的双向流式会话
// 参数:
//   - stream: gRPC 双向流
//
// 返回值:
//   - error: 会话过程中的错误
//
// 执行流程:
//   - 持续接收来自客户端的消息
//   - 根据消息类型分发到对应的处理器
//   - 当接收到 EOF 或错误时退出
func (cps *computeService) Session(stream grpc.BidiStreamingServer[cluster.Message, cluster.Message]) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		cps.onReceive(stream, msg)
	}
}

// onReceive 处理接收到的计算消息
// 参数:
//   - stream: gRPC 双向流
//   - msg: 计算消息
//
// 消息类型处理:
//   - Ready: 设置计算流的发送函数
//   - 其他: 将消息转发到对应的计算流
func (cps *computeService) onReceive(stream grpc.BidiStreamingServer[cluster.Message, cluster.Message], msg *cluster.Message) {
	cps.mu.RLock()
	c, ok := cps.computers[msg.ConnID]
	cps.mu.RUnlock()

	if !ok {
		logrus.Errorf("compute session %s not found", msg.ConnID)
		return
	}

	switch msg.Message.(type) {
	case *cluster.Message_Ready:
		c.SetSender(stream.Send)
	default:
		c.Produce(msg)
	}
}

// newConn 创建或获取一个计算流实例
// 参数:
//   - ctx: 上下文，用于控制计算流生命周期
//   - connId: 连接标识符
//
// 返回值:
//   - *transport.ComputeStreamImpl: 计算流实例
//
// 注意: 所有计算流会在 close 方法中统一清理
func (cps *computeService) newConn(ctx context.Context, connId string) *transport.ComputeStreamImpl {
	cps.mu.RLock()
	c, ok := cps.computers[connId]
	cps.mu.RUnlock()

	if ok {
		return c
	}

	cps.mu.Lock()
	defer cps.mu.Unlock()

	// 双重检查
	if c, ok := cps.computers[connId]; ok {
		return c
	}

	c = transport.NewComputeStreamImpl(connId, transport.RPC)
	go c.Run(ctx)
	cps.computers[connId] = c
	return c
}

// close 关闭计算服务并清理所有连接
func (cps *computeService) close() {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	for _, c := range cps.computers {
		_ = c.Close()
	}
	// 清空 map，释放内存
	cps.computers = make(map[string]*transport.ComputeStreamImpl)
}
