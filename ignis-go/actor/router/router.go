// Package router 提供了基于 Proto Actor 的消息路由功能
// 支持将消息根据目标 ID 路由到对应的 Actor PID
package router

import (
	"log/slog"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

// Context is a hack for `actor.Context` and `*actor.RootContext`, since these two types are not
// compatible.
type Context interface {
	Send(pid *actor.PID, msg any)
	Logger() *slog.Logger
}

type Router interface {
	Send(targetId string, msg any)
	Register(targetId string, pid *actor.PID)
	Deregister(targetId string)
}

type baseRouter struct {
	mu     sync.RWMutex // 读写锁，保护路由表和默认目标
	ctx    Context
	routes map[string]*actor.PID // 路由表，key 为目标 ID
}

// Register 注册目标 Actor
// 参数:
//   - targetId: 目标 Actor 标识符
//   - pid: Actor 进程 ID
//
// 如果目标已存在，会覆盖旧的 PID
func (r *baseRouter) Register(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[targetId] = pid
}

// Unregister 注销目标 Actor
// 参数:
//   - targetId: 目标 Actor 标识符
func (r *baseRouter) Deregister(targetId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, targetId)
}

func makeBaseRouter(ctx Context) baseRouter {
	return baseRouter{
		ctx:    ctx,
		routes: make(map[string]*actor.PID),
	}
}
