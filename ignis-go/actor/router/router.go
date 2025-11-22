// Package router 提供了基于 Proto Actor 的消息路由功能
// 支持将消息根据目标 ID 路由到对应的 Actor PID
package router

import (
	"log/slog"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"
)

// Context 是路由器使用的上下文接口
// 提供消息发送和日志记录功能
type Context interface {
	// Send 发送消息到指定的 Actor PID
	Send(pid *actor.PID, msg any)

	// Logger 返回日志记录器
	Logger() *slog.Logger
}

// DefaultRouter 是全局默认路由器实例
// 可直接使用包级别函数操作该路由器
var DefaultRouter = NewRouter()

// Send 通过默认路由器发送消息
// 参数:
//   - ctx: 路由上下文
//   - targetId: 目标 Actor 标识符
//   - msg: 要发送的消息
func Send(ctx Context, targetId string, msg any) {
	DefaultRouter.Send(ctx, targetId, msg)
}

// Register 在默认路由器中注册目标
// 参数:
//   - targetId: 目标 Actor 标识符
//   - pid: Actor 进程 ID
func Register(targetId string, pid *actor.PID) {
	DefaultRouter.Register(targetId, pid)
}

// Unregister 从默认路由器中注销目标
// 参数:
//   - targetId: 目标 Actor 标识符
func Unregister(targetId string) {
	DefaultRouter.Unregister(targetId)
}

// RegisterIfAbsent 在默认路由器中注册目标（如果不存在）
// 参数:
//   - targetId: 目标 Actor 标识符
//   - pid: Actor 进程 ID
func RegisterIfAbsent(targetId string, pid *actor.PID) {
	DefaultRouter.RegisterIfAbsent(targetId, pid)
}

// SetDefaultTarget 设置默认路由器的默认目标
// 参数:
//   - pid: 默认目标的 Actor 进程 ID
func SetDefaultTarget(pid *actor.PID) {
	DefaultRouter.SetDefaultTarget(pid)
}

// Router 是消息路由器
// 根据目标 ID 查找并路由消息到对应的 Actor PID
//
// 功能:
//   - 维护 targetId 到 PID 的映射表
//   - 支持设置默认目标（当找不到目标时使用）
//   - 线程安全的路由操作
type Router struct {
	mu            sync.RWMutex          // 读写锁，保护路由表和默认目标
	routeTable    map[string]*actor.PID // 路由表，key 为目标 ID
	defaultTarget *actor.PID            // 默认目标 PID
}

// NewRouter 创建一个新的路由器实例
// 返回值:
//   - *Router: 路由器实例
func NewRouter() *Router {
	return &Router{
		routeTable: make(map[string]*actor.PID),
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
func (r *Router) Send(ctx Context, targetId string, msg any) {
	r.mu.RLock()
	pid, ok := r.routeTable[targetId]
	defaultPid := r.defaultTarget
	r.mu.RUnlock()

	if !ok {
		if defaultPid == nil {
			ctx.Logger().Error("target not found", "targetId", targetId)
			return
		}
		ctx.Logger().Info("use default target", "targetId", targetId, "default target pid", defaultPid)
		pid = defaultPid
	}

	ctx.Send(pid, msg)
}

// SetDefaultTarget 设置默认目标 PID
// 参数:
//   - pid: 默认目标的 Actor 进程 ID
//
// 当路由表中找不到目标时，会使用默认目标
func (r *Router) SetDefaultTarget(pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultTarget = pid
	logrus.Info("set default target", "pid", pid)
}

// Register 注册目标 Actor
// 参数:
//   - targetId: 目标 Actor 标识符
//   - pid: Actor 进程 ID
//
// 如果目标已存在，会覆盖旧的 PID
func (r *Router) Register(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routeTable[targetId] = pid
}

// Unregister 注销目标 Actor
// 参数:
//   - targetId: 目标 Actor 标识符
func (r *Router) Unregister(targetId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routeTable, targetId)
}

// RegisterIfAbsent 注册目标 Actor（仅当不存在时）
// 参数:
//   - targetId: 目标 Actor 标识符
//   - pid: Actor 进程 ID
//
// 如果目标已存在，不会覆盖，直接返回
// TODO: 根据消息最短路径，动态更新路由表
func (r *Router) RegisterIfAbsent(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.routeTable[targetId]; ok {
		return
	}

	r.routeTable[targetId] = pid
}
