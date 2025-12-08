// Package task 提供了任务节点和运行时的抽象
// 支持基于函数、Actor、ActorGroup 的不同任务封装方式
package task

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/proto"
)

// Node 表示任务 DAG 中的一个节点
// 每个节点封装了一个可执行的任务单元
//
// 节点包含:
//   - id: 节点的唯一标识符
//   - inputs: 节点的输入依赖（参数名列表）
//   - producer: 处理器生产者，用于创建实际的任务处理器
//
// Node 可以从多种来源创建:
//   - Function: 封装一个函数调用
//   - Actor PID: 封装一个已存在的 Actor
//   - ActorGroup: 封装一组 Actor（用于负载均衡或并行）
type Node struct {
	id       string          // 节点唯一标识符
	inputs   []string        // 输入依赖列表（参数名）
	producer HandlerProducer // 处理器生产者
}

// Runtime 创建节点的运行时实例
// 参数:
//   - sessionId: 会话 ID
//   - store: Store Actor 的 PID
//   - replyTo: 回复目标标识符
//
// 返回值:
//   - *Runtime: 运行时实例
//
// 运行时实例用于实际执行节点任务，包含就绪条件和处理器
func (node *Node) Runtime(sessionId string, store *actor.PID, replyTo string) *Runtime {
	return &Runtime{
		replyTo: replyTo,
		handler: node.producer(sessionId, store),
		cond:    sync.NewCond(&sync.Mutex{}),
	}
}

// NewNode 创建一个新的任务节点
// 参数:
//   - id: 节点 ID
//   - inputs: 输入参数名列表
//   - handler: 处理器生产者
//
// 返回值:
//   - *Node: 节点实例
func NewNode(id string, inputs []string, handler HandlerProducer) *Node {
	return &Node{
		id:       id,
		inputs:   inputs,
		producer: handler,
	}
}

// NodeFromFunction 从函数创建任务节点
// 参数:
//   - id: 节点 ID
//   - f: 函数实例（可以是 GoFunction 或 PyFunction）
//
// 返回值:
//   - *Node: 节点实例
//
// 该方法会自动从函数中提取参数列表作为输入依赖
// 适用于封装单个函数调用的任务
func NodeFromFunction(id string, f functions.Function) *Node {
	return NewNode(id, f.Params(), ProducerFromFunction(f))
}

// NodeFromPID 从 Actor PID 创建任务节点
// 参数:
//   - id: 节点 ID
//   - params: 输入参数名列表
//   - pid: Actor 的 PID
//
// 返回值:
//   - *Node: 节点实例
//
// 适用于封装已存在的 Actor 的任务
// Actor 需要能够处理参数并返回结果
func NodeFromPID(id string, params []string, pid *actor.PID) *Node {
	return NewNode(id, params, ProducerFromPID(params, pid))
}

// NodeFromActorGroup 从 Actor 组创建任务节点
// 参数:
//   - id: 节点 ID
//   - params: 输入参数名列表
//   - group: Actor 组
//
// 返回值:
//   - *Node: 节点实例
//
// 适用于封装多个 Actor 协同工作的任务
// 可用于负载均衡或并行处理
func NodeFromActorGroup(id string, params []string, group *ActorGroup) *Node {
	return NewNode(id, params, ProducerFromActorGroup(params, group))
}

// Runtime 是节点的运行时实例
// 负责实际执行节点任务，管理依赖就绪状态和任务启动
//
// 工作原理:
//   - handler: 实际的任务处理器
//   - cond: 条件变量，用于等待所有依赖就绪
//   - replyTo: 任务完成后的回复目标
//
// 执行流程:
//  1. 通过 Invoke 方法接收输入参数
//  2. 当所有依赖就绪时，条件变量发出信号
//  3. Start 方法等待就绪信号，然后启动任务
type Runtime struct {
	replyTo string     // 回复目标标识符
	handler Handler    // 任务处理器
	cond    *sync.Cond // 条件变量，用于等待依赖就绪
}

// Start 启动节点任务
// 参数:
//   - ctx: Actor 上下文
//
// 返回值:
//   - error: 启动或执行错误
//
// 该方法会阻塞等待直到所有依赖就绪（通过条件变量），然后启动处理器
func (rt *Runtime) Start(ctx actor.Context) error {
	rt.cond.L.Lock()
	for !rt.handler.Ready() {
		rt.cond.Wait()
	}
	rt.cond.L.Unlock()

	return rt.handler.Start(ctx, rt.replyTo)
}

// Invoke 向节点提供输入参数
// 参数:
//   - ctx: Actor 上下文
//   - param: 参数名
//   - value: 参数值（Flow 引用）
//
// 返回值:
//   - error: 调用错误
//
// 执行流程:
//  1. 调用处理器的 Invoke 方法设置参数
//  2. 如果所有参数已就绪，发出条件变量信号通知 Start 方法
//
// 注意: 该方法可能被多次调用，每次提供一个参数
func (rt *Runtime) Invoke(ctx actor.Context, param string, value *proto.Flow) (err error) {
	ready, err := rt.handler.Invoke(ctx, param, value)
	if err != nil {
		return err
	}

	if ready {
		rt.cond.L.Lock()
		rt.cond.Signal()
		rt.cond.L.Unlock()
	}

	return
}
