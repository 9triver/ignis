// Package rpc 提供基于 gRPC 的远程过程调用（RPC）传输实现
package rpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/transport"
	"github.com/sirupsen/logrus"
)

const (
	// gracefulShutdownTimeout 是优雅关闭的最大等待时间
	gracefulShutdownTimeout = 10 * time.Second
)

// ConnectionManager 是基于 gRPC 的 RPC 连接管理器
// 负责管理多个 gRPC 服务和它们的连接
//
// 包含的服务:
//   - controllerService: 控制器服务，处理控制器之间的通信
//   - executorService: 执行器服务，处理 Python 执行器的通信
//   - computeService: 计算服务，处理集群计算节点的通信
type ConnectionManager struct {
	addr string             // gRPC 服务器监听地址
	cs   *controllerService // 控制器服务实例
	es   *executorService   // 执行器服务实例
	cps  *computeService    // 计算服务实例
}

// Addr 返回 gRPC 服务器的监听地址
func (cm *ConnectionManager) Addr() string {
	return cm.addr
}

// Run 启动 gRPC 服务器并阻塞运行
// 参数:
//   - ctx: 上下文，用于控制生命周期
//
// 返回值:
//   - error: 运行过程中的错误
//
// 执行流程:
//  1. 监听 TCP 端口
//  2. 创建 gRPC 服务器（配置 512MB 消息大小限制）
//  3. 注册控制器和执行器服务
//  4. 在 goroutine 中启动服务器
//  5. 等待 context 取消或服务器错误
//  6. 通过 defer 自动清理所有服务和连接
//
// 注意: 该方法会阻塞，应在独立的 goroutine 中调用
func (cm *ConnectionManager) Run(ctx context.Context) error {
	defer cm.cs.close()
	defer cm.es.close()
	defer cm.cps.close()

	lis, err := net.Listen("tcp", cm.addr)
	if err != nil {
		return err
	}
	logrus.Infof("RPC server listening on %s", cm.addr)

	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(512*1024*1024),
		grpc.MaxSendMsgSize(512*1024*1024),
	)

	// 优雅关闭 gRPC 服务器
	defer func() {
		// 启动优雅关闭
		done := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(done)
		}()

		// 等待优雅关闭完成或超时
		select {
		case <-done:
			logrus.Info("gRPC server gracefully stopped")
		case <-time.After(gracefulShutdownTimeout):
			logrus.Warn("gRPC server graceful shutdown timeout, forcing stop")
			server.Stop()
		}
	}()

	// 注册 gRPC 服务
	controller.RegisterServiceServer(server, cm.cs)
	executor.RegisterServiceServer(server, cm.es)
	// cluster.RegisterServiceServer(server, cm.cps)

	ech := make(chan error)
	go func() {
		ech <- server.Serve(lis)
		close(ech)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ech:
		return err
	}
}

// NextController 获取下一个可用的控制器连接
// 返回值:
//   - transport.Controller: 控制器流实例
//
// 该方法会阻塞直到有新的控制器连接
func (cm *ConnectionManager) NextController() transport.Controller {
	return cm.cs.nextConn()
}

// NewExecutor 创建或获取一个执行器实例
// 参数:
//   - ctx: 上下文，用于控制执行器生命周期
//   - conn: 连接标识符
//
// 返回值:
//   - transport.Executor: 执行器实例
func (cm *ConnectionManager) NewExecutor(ctx context.Context, conn string) transport.Executor {
	return cm.es.newConn(ctx, conn)
}

// Type 返回传输协议类型
func (cm *ConnectionManager) Type() transport.Protocol {
	return transport.RPC
}

// NewManager 创建一个新的 RPC 连接管理器
// 参数:
//   - addr: gRPC 服务器监听地址（例如：:50051）
//
// 返回值:
//   - *ConnectionManager: 管理器实例
func NewManager(addr string) *ConnectionManager {
	return &ConnectionManager{
		addr: addr,
		cs: &controllerService{
			controllers: make(map[string]*transport.ControllerImpl),
			next:        make(chan *transport.ControllerImpl),
		},
		es:  &executorService{executors: make(map[string]*transport.ExecutorImpl)},
		cps: &computeService{computers: make(map[string]*transport.ComputeStreamImpl)},
	}
}
