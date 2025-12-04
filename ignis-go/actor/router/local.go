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
	router := &LocalRouter{
		baseRouter: makeBaseRouter(ctx),
	}

	return router
}
