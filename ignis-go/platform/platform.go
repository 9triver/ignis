// Package platform 提供了 Ignis 平台的核心实现
// 负责管理应用生命周期、任务调度、通信管理和状态监控
package platform

import (
	"context"
	"path"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/transport/ipc"
	"github.com/9triver/ignis/transport/rpc"
	"github.com/9triver/ignis/utils"
)

// Platform 是 Ignis 平台的核心结构
// 负责管理整个平台的运行，包括：
//   - Actor 系统管理
//   - 控制器和执行器连接管理
//   - 任务部署和调度
type Platform struct {
	ctx context.Context             // 平台根上下文
	sys *actor.ActorSystem          // Actor 系统
	cm  transport.ControllerManager // 控制器连接管理器
	em  transport.ExecutorManager   // 执行器连接管理器
	dp  task.Deployer               // 任务部署器
}

// Run 启动平台并阻塞运行
// 返回值:
//   - error: 运行过程中的错误
//
// 执行流程:
//  1. 启动控制器管理器（处理客户端连接）
//  2. 启动执行器管理器（处理 Python 执行器连接）
//  3. 启动客户端连接处理循环（每个连接启动一个 controller）
//  4. 等待 context 取消
//
// 注意: 该方法会阻塞直到 context 取消
func (p *Platform) Run() error {
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	// 启动控制器管理器
	go func() {
		logrus.Infof("Controller Manager listening on %s", p.cm.Addr())
		if err := p.cm.Run(ctx); err != nil {
			logrus.Errorf("Controller Manager error: %v", err)
			cancel()
		}
	}()

	// 启动执行器管理器
	go func() {
		logrus.Infof("Executor Manager listening on %s", p.em.Addr())
		if err := p.em.Run(ctx); err != nil {
			logrus.Errorf("Executor Manager error: %v", err)
			cancel()
		}
	}()

	// 启动客户端连接处理循环
	go p.handleClientConnections(ctx)

	<-ctx.Done()
	return ctx.Err()
}

// handleClientConnections 处理客户端连接
// 参数:
//   - ctx: 上下文
//
// 该方法持续监听新的控制器连接，每个连接启动一个 controller
func (p *Platform) handleClientConnections(ctx context.Context) {
	logrus.Info("Waiting for new client connection...")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ctrlr := p.cm.NextController()
		if ctrlr == nil {
			continue
		}

		logrus.Info("New client is connected, starting controller...")
		// 为每个连接生成唯一的 session ID
		sessionID := utils.GenIDWith("session-")

		control.SpawnTaskControllerV2(p.sys.Root, sessionID, p.dp, ctrlr, func() {
			logrus.Infof("Controller for session %s is closed", sessionID)
		})

		logrus.Infof("Controller started for session %s", sessionID)
	}
}

// NewPlatform 创建一个新的平台实例
// 参数:
//   - ctx: 平台根上下文
//   - rpcAddr: RPC 服务器监听地址（用于客户端连接）
//   - dp: 任务部署器（如果为 nil，会创建默认的虚拟环境部署器）
//
// 返回值:
//   - *Platform: 平台实例
//
// 初始化内容:
//  1. 创建 Actor 系统
//  2. 创建 IPC 执行器管理器（用于 Python 执行器）
//  3. 创建 RPC 控制器管理器（用于客户端）
//  4. 如果未提供部署器，创建默认的虚拟环境管理器
func NewPlatform(ctx context.Context, rpcAddr string, dp task.Deployer) *Platform {
	opt := utils.WithLogger()
	ipcAddr := "ipc://" + path.Join(configs.StoragePath, "em-ipc")

	sys := actor.NewActorSystem(opt)
	em := ipc.NewManager(ipcAddr)

	if dp == nil {
		vm, err := python.NewVenvManager(ctx, em)
		if err != nil {
			panic(err)
		}
		dp = task.NewVenvMgrDeployer(vm)
	}

	return &Platform{
		ctx: ctx,
		sys: sys,
		cm:  rpc.NewManager(rpcAddr),
		em:  em,
		dp:  dp,
	}
}
