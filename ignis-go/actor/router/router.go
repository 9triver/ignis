// Package router 提供了基于 Proto Actor 的消息路由功能
// 支持将消息根据目标 ID 路由到对应的 Actor PID
package router

import (
	"sync"

	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

// Context is a hack for `actor.Context` and `*actor.RootContext`, since these two types are not
// compatible.
type Context interface {
	actor.SenderContext
	actor.SpawnerContext
}

type Router interface {
	Send(targetId string, msg any)
	Register(store *proto.StoreRef)
	RegisterActor(targetId string, pid *actor.PID) // 注册普通的 actor
	Deregister(targetId string)
	Shutdown()
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
// func (r *baseRouter) Register(targetId string, pid *actor.PID) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()

// 	r.routes[targetId] = pid
// }

func (r *baseRouter) Register(store *proto.StoreRef) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[store.ID] = store.PID
	r.ctx.Logger().Info("router: registered store", "id", store.ID, "pid", store.PID, "total_routes", len(r.routes))
}

// RegisterActor 注册普通的 actor 到路由表
func (r *baseRouter) RegisterActor(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[targetId] = pid
	r.ctx.Logger().Info("router: registered actor", "id", targetId, "pid", pid, "total_routes", len(r.routes))
}

// Unregister 注销目标 Actor
// 参数:
//   - targetId: 目标 Actor 标识符
func (r *baseRouter) Deregister(targetId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, targetId)
}

func (r *baseRouter) Send(targetId string, msg any) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pid, ok := r.routes[targetId]
	if !ok {
		r.ctx.Logger().Warn("router: target not found in routes", "target", targetId, "available_routes", len(r.routes))
		// 打印所有可用的路由键以便调试
		routeKeys := make([]string, 0, len(r.routes))
		for k := range r.routes {
			routeKeys = append(routeKeys, k)
		}
		r.ctx.Logger().Info("router: available routes", "count", len(r.routes), "routes", routeKeys)
		return
	}

	r.ctx.Logger().Info("router: sending message to target", "target", targetId, "pid", pid)
	r.ctx.Send(pid, msg)
}

func (r *baseRouter) Shutdown() {}

func makeBaseRouter(ctx Context) baseRouter {
	return baseRouter{
		ctx:    ctx,
		routes: make(map[string]*actor.PID),
	}
}
