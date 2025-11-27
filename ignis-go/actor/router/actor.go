package router

import (
	"github.com/asynkron/protoactor-go/actor"
)

// ActorRouter 是消息路由器
// 根据目标 ID 查找并路由消息到对应的 Actor PID
//
// 功能:
//   - 维护 targetId 到 PID 的映射表
//   - 支持设置默认目标（当找不到目标时使用）
//   - 线程安全的路由操作
type ActorRouter struct {
	baseRouter
}

// NewActorRouter 创建一个新的路由器实例
// 返回值:
//   - *Router: 路由器实例
func NewActorRouter(ctx Context) *ActorRouter {
	return &ActorRouter{
		baseRouter: baseRouter{
			ctx:        ctx,
			routeTable: make(map[string]*actor.PID),
		},
	}
}

// Send 发送消息到指定的目标 Actor
// 参数:
//   - ctx: 路由上下文
//   - targetId: 目标 Actor 标识符
//   - msg: 要发送的消息
//
// 路由逻辑:
//   - 如果找到目标，发送到对应的 PID
//   - 如果找不到目标且有默认目标，发送到默认目标
//   - 如果都没有，记录错误日志
func (r *ActorRouter) Send(targetId string, msg any) {
	r.mu.RLock()
	pid, ok := r.routeTable[targetId]
	defaultPid := r.defaultTarget
	r.mu.RUnlock()

	if !ok {
		if defaultPid == nil {
			r.ctx.Logger().Error("target not found", "targetId", targetId)
			return
		}
		r.ctx.Logger().Info("use default target", "targetId", targetId, "default target pid", defaultPid)
		pid = defaultPid
	}

	r.ctx.Send(pid, msg)
}
