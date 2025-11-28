package router

// LocalRouter 是消息路由器
// 根据目标 ID 查找并路由消息到对应的 Actor PID
//
// 功能:
//   - 维护 targetId 到 PID 的映射表
//   - 支持设置默认目标（当找不到目标时使用）
//   - 线程安全的路由操作
type LocalRouter struct {
	baseRouter
}

// NewLocalRouter 创建一个新的路由器实例
// 返回值:
//   - *Router: 路由器实例
func NewLocalRouter(ctx Context) *LocalRouter {
	return &LocalRouter{
		baseRouter: makeBaseRouter(ctx),
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
func (r *LocalRouter) Send(targetId string, msg any) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pid, ok := r.routes[targetId]
	if !ok {
		return
	}

	r.ctx.Send(pid, msg)
}
