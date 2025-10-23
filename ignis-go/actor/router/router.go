package router

import (
	"log/slog"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"
)

type Context interface {
	Send(pid *actor.PID, msg any)
	Logger() *slog.Logger
}

var DefaultRouter = NewRouter()

func Send(ctx Context, targetId string, msg any) {
	DefaultRouter.Send(ctx, targetId, msg)
}

func Register(targetId string, pid *actor.PID) {
	DefaultRouter.Register(targetId, pid)
}

func Unregister(targetId string) {
	DefaultRouter.Unregister(targetId)
}

func RegisterIfAbsent(targetId string, pid *actor.PID) {
	DefaultRouter.RegisterIfAbsent(targetId, pid)
}

func SetDefaultTarget(pid *actor.PID) {
	DefaultRouter.SetDefaultTarget(pid)
}

type Router struct {
	mu            sync.Mutex
	routeTable    map[string]*actor.PID
	defaultTarget *actor.PID
}

func NewRouter() *Router {
	return &Router{
		routeTable: make(map[string]*actor.PID),
	}
}

func (r *Router) Send(ctx Context, targetId string, msg any) {
	r.mu.Lock() // TODO: use read lock
	defer r.mu.Unlock()

	pid, ok := r.routeTable[targetId]
	if !ok {
		if r.defaultTarget == nil {
			ctx.Logger().Error("target not found", "targetId", targetId)
			return
		} else {
			ctx.Logger().Info("use default target", "targetId", targetId, "default target pid", r.defaultTarget)
			pid = r.defaultTarget
		}
	}

	ctx.Logger().Debug("route message to target", "targetId", targetId, "pid", pid, "msg", msg)
	ctx.Send(pid, msg)
}

func (r *Router) SetDefaultTarget(pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultTarget = pid
	logrus.Info("set default target", "pid", pid)
}

func (r *Router) Register(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routeTable[targetId] = pid
}

func (r *Router) Unregister(targetId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routeTable, targetId)
}

// TODO: 根据消息最短路径，动态更新路由表
func (r *Router) RegisterIfAbsent(targetId string, pid *actor.PID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.routeTable[targetId]; ok {
		return
	}

	r.routeTable[targetId] = pid
}
