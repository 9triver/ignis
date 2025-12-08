// Package platform 提供了 Ignis 平台的核心实现
// 负责管理应用生命周期、任务调度、通信管理和状态监控
package platform

import (
	"context"
	"path"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/transport/ipc"
	"github.com/9triver/ignis/transport/rpc"
	"github.com/9triver/ignis/utils"
)

// Platform 是 Ignis 平台的核心结构
// 负责管理整个平台的运行，包括：
//   - Actor 系统管理
//   - 控制器和执行器连接管理
//   - 应用注册和生命周期管理
//   - 任务部署和调度
type Platform struct {
	ctx                 context.Context             // 平台根上下文
	sys                 *actor.ActorSystem          // Actor 系统
	cm                  transport.ControllerManager // 控制器连接管理器
	em                  transport.ExecutorManager   // 执行器连接管理器
	dp                  task.Deployer               // 任务部署器
	appInfos            map[string]*ApplicationInfo // 应用信息映射表
	controllerActorRefs map[string]*proto.ActorRef  // 控制器 Actor 引用映射表
	mu                  sync.RWMutex                // 保护应用信息的读写锁
}

// Run 启动平台并阻塞运行
// 返回值:
//   - error: 运行过程中的错误
//
// 执行流程:
//  1. 启动控制器管理器（处理客户端连接）
//  2. 启动执行器管理器（处理 Python 执行器连接）
//  3. 启动客户端注册循环（处理应用注册请求）
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

	// 启动客户端注册循环
	go p.handleClientRegistrations(ctx)

	<-ctx.Done()
	return ctx.Err()
}

// handleClientRegistrations 处理客户端注册请求
// 参数:
//   - ctx: 上下文
//
// 该方法持续监听新的控制器连接，处理应用注册请求
func (p *Platform) handleClientRegistrations(ctx context.Context) {
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

		logrus.Info("New client is connected")
		if err := p.registerApplication(ctrlr); err != nil {
			logrus.Errorf("Failed to register application: %v", err)
		}
	}
}

// registerApplication 处理单个应用的注册
// 参数:
//   - ctrlr: 控制器连接
//
// 返回值:
//   - error: 注册错误
func (p *Platform) registerApplication(ctrlr transport.Controller) error {
	msg := <-ctrlr.RecvChan()

	if msg.Type != controller.CommandType_FR_REGISTER_REQUEST {
		logrus.Errorf("The first message %s is not register request", msg.Type)
		return nil
	}

	req := msg.GetRegisterRequest()
	if req == nil {
		logrus.Error("Register request is nil")
		return nil
	}

	appID := req.GetApplicationID()

	// 检查应用 ID 是否冲突
	p.mu.RLock()
	_, exists := p.appInfos[appID]
	p.mu.RUnlock()

	if exists {
		logrus.Errorf("Application ID %s is conflicted", appID)
		return nil
	}

	// 创建应用信息和控制器 Actor
	appInfo := NewApplicationInfo(appID)
	actorRef := control.SpawnTaskControllerV2(p.sys.Root, appID, p.dp, appInfo, ctrlr, func() {
		// 清理应用信息
		p.mu.Lock()
		delete(p.appInfos, appID)
		delete(p.controllerActorRefs, appID)
		p.mu.Unlock()
	})

	p.mu.Lock()
	p.appInfos[appID] = appInfo
	p.controllerActorRefs[appID] = actorRef
	p.mu.Unlock()

	logrus.Infof("Application %s is registered", appID)

	// 发送确认消息
	ack := controller.NewAck(nil)
	ctrlr.SendChan() <- ack

	return nil
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
		ctx:                 ctx,
		sys:                 sys,
		cm:                  rpc.NewManager(rpcAddr),
		em:                  em,
		dp:                  dp,
		appInfos:            make(map[string]*ApplicationInfo),
		controllerActorRefs: make(map[string]*proto.ActorRef),
	}
}

// GetApplicationInfo 获取指定应用的信息
// 参数:
//   - appID: 应用标识符
//
// 返回值:
//   - *ApplicationInfo: 应用信息，如果应用不存在则返回 nil
func (p *Platform) GetApplicationInfo(appID string) *ApplicationInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.appInfos[appID]
}

// ApplicationInfo 存储应用的信息和状态
type ApplicationInfo struct {
	ID    string       // 应用 ID
	mutex sync.RWMutex // 保护并发访问的读写锁
}

// NewApplicationInfo 创建一个新的应用信息实例
// 参数:
//   - appID: 应用标识符
//
// 返回值:
//   - *ApplicationInfo: 应用信息实例
func NewApplicationInfo(appID string) *ApplicationInfo {
	return &ApplicationInfo{
		ID: appID,
	}
}
