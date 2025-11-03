// Package platform 提供了 Ignis 平台的核心实现
// 负责管理应用生命周期、任务调度、通信管理和状态监控
package platform

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions"
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
		vm, err := functions.NewVenvManager(ctx, em)
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

// GetApplicationDAG 获取指定应用的 DAG（有向无环图）
// 参数:
//   - appID: 应用标识符
//
// 返回值:
//   - *controller.DAG: 应用的 DAG，如果应用不存在则返回 nil
func (p *Platform) GetApplicationDAG(appID string) *controller.DAG {
	logrus.Infof("GetApplicationDAG: %s", appID)

	p.mu.RLock()
	appInfo, ok := p.appInfos[appID]
	p.mu.RUnlock()

	if !ok {
		return nil
	}
	return appInfo.GetDAG()
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

// NodeState 表示 DAG 节点的状态
type NodeState struct {
	ID       string    `json:"id"`       // 节点 ID
	Type     string    `json:"type"`     // 节点类型: "control" 或 "data"
	Done     bool      `json:"done"`     // 是否已完成
	Ready    bool      `json:"ready"`    // 是否已就绪
	UpdateAt time.Time `json:"updateAt"` // 最后更新时间
}

// DAGStateChangeEvent 表示 DAG 状态变更事件
// 当 DAG 中的节点状态发生变化时，会生成该事件并通知观察者
type DAGStateChangeEvent struct {
	AppID     string     `json:"appId"`     // 应用 ID
	NodeID    string     `json:"nodeId"`    // 节点 ID
	NodeState *NodeState `json:"nodeState"` // 节点状态
	Timestamp time.Time  `json:"timestamp"` // 事件时间戳
}

// StateChangeObserver 定义状态变更观察者接口
// 实现该接口可以监听 DAG 状态变化
type StateChangeObserver interface {
	// OnDAGStateChanged 当 DAG 状态变化时调用
	OnDAGStateChanged(event *DAGStateChangeEvent)
}

// ApplicationInfo 存储应用的信息和状态
// 包括 DAG、节点状态、观察者列表等
type ApplicationInfo struct {
	ID         string                // 应用 ID
	dag        *controller.DAG       // 应用的 DAG
	nodeStates map[string]*NodeState // 节点状态映射表
	observers  []StateChangeObserver // 状态变更观察者列表
	mutex      sync.RWMutex          // 保护并发访问的读写锁
}

// NewApplicationInfo 创建一个新的应用信息实例
// 参数:
//   - appID: 应用标识符
//
// 返回值:
//   - *ApplicationInfo: 应用信息实例
func NewApplicationInfo(appID string) *ApplicationInfo {
	return &ApplicationInfo{
		ID:         appID,
		nodeStates: make(map[string]*NodeState),
		observers:  make([]StateChangeObserver, 0),
	}
}

// SetDAG 设置应用的 DAG 并初始化节点状态
// 参数:
//   - dag: DAG 对象
//
// 该方法会：
//  1. 保存 DAG 引用
//  2. 从 DAG 中提取所有节点
//  3. 为每个节点创建初始状态
//  4. 区分控制节点和数据节点进行不同的初始化
func (a *ApplicationInfo) SetDAG(dag *controller.DAG) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.dag = dag

	// Initialize node states from DAG
	if a.nodeStates == nil {
		a.nodeStates = make(map[string]*NodeState)
	}

	for _, node := range dag.Nodes {
		var nodeState *NodeState
		if node.Type == "ControlNode" && node.GetControlNode() != nil {
			cn := node.GetControlNode()
			nodeState = &NodeState{
				ID:       cn.Id,
				Type:     "control",
				Done:     cn.Done,
				Ready:    true, // Control nodes are ready by default
				UpdateAt: time.Now(),
			}
		} else if node.Type == "DataNode" && node.GetDataNode() != nil {
			dn := node.GetDataNode()
			nodeState = &NodeState{
				ID:       dn.Id,
				Type:     "data",
				Done:     dn.Done,
				Ready:    dn.Ready,
				UpdateAt: time.Now(),
			}
		}

		if nodeState != nil {
			a.nodeStates[nodeState.ID] = nodeState
		}
	}
}

// GetDAG 获取应用的 DAG
// 返回值:
//   - *controller.DAG: DAG 对象
//
// 该方法是线程安全的
func (a *ApplicationInfo) GetDAG() *controller.DAG {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.dag
}

// MarkNodeDone 标记节点为已完成并通知观察者
// 参数:
//   - nodeID: 节点 ID
//
// 执行操作:
//  1. 更新节点状态为 Done
//  2. 同步更新 DAG 中的节点状态
//  3. 生成状态变更事件
//  4. 异步通知所有观察者
func (a *ApplicationInfo) MarkNodeDone(nodeID string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if nodeState, exists := a.nodeStates[nodeID]; exists {
		nodeState.Done = true
		nodeState.UpdateAt = time.Now()

		// Update the DAG node state as well
		if a.dag != nil {
			for _, node := range a.dag.Nodes {
				if node.Type == "ControlNode" && node.GetControlNode() != nil && node.GetControlNode().Id == nodeID {
					node.GetControlNode().Done = true
				} else if node.Type == "DataNode" && node.GetDataNode() != nil && node.GetDataNode().Id == nodeID {
					node.GetDataNode().Done = true
				}
			}
		}

		// Notify observers
		event := &DAGStateChangeEvent{
			AppID:     a.ID,
			NodeID:    nodeID,
			NodeState: nodeState,
			Timestamp: time.Now(),
		}

		for _, observer := range a.observers {
			go observer.OnDAGStateChanged(event)
		}
	}
}

// GetNodeStates 获取所有节点的状态
// 返回值:
//   - map[string]*NodeState: 节点状态映射表的副本
//
// 该方法返回副本以避免并发访问问题，是线程安全的
func (a *ApplicationInfo) GetNodeStates() map[string]*NodeState {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// 返回副本以避免竞态条件
	states := make(map[string]*NodeState)
	for k, v := range a.nodeStates {
		stateCopy := *v
		states[k] = &stateCopy
	}
	return states
}

// AddObserver 添加状态变更观察者
// 参数:
//   - observer: 观察者实例
//
// 观察者会在节点状态变化时收到通知
func (a *ApplicationInfo) AddObserver(observer StateChangeObserver) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.observers = append(a.observers, observer)
}

// RemoveObserver 移除状态变更观察者
// 参数:
//   - observer: 要移除的观察者实例
func (a *ApplicationInfo) RemoveObserver(observer StateChangeObserver) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for i, obs := range a.observers {
		if obs == observer {
			a.observers = append(a.observers[:i], a.observers[i+1:]...)
			break
		}
	}
}
