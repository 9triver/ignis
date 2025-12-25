// Package store 提供了分布式对象存储的 Actor 实现
package store

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/cache"
	"github.com/9triver/ignis/utils/errors"
)

// Actor 是分布式对象存储 Actor
// 负责管理本地对象和协调远程对象的访问
//
// 工作原理:
//   - localObjects: 存储本地生成的对象
//   - remoteObjects: 管理对远程对象的请求（使用 Future 模式）
//   - 支持对象的保存、获取、流式传输
//   - 通过路由器与其他 Store Actor 通信
type Actor struct {
	mu            sync.RWMutex
	id            string                                     // Store 标识符
	ref           *proto.StoreRef                            // Store 引用（包含 ID 和 PID）
	router        router.Router                              // 传输存根（用于远程通信）
	localObjects  map[string]object.Interface                // 本地对象映射表
	remoteObjects map[string]utils.Future[object.Interface]  // 远程对象 Future 映射表
	cache         cache.Controller[string, object.Interface] // 对象缓存管理
}

// onObjectRequest 处理来自远程 Store 的对象请求
// 参数:
//   - ctx: Actor 上下文
//   - req: 对象请求消息
//
// 执行流程:
//  1. 验证请求目标是否为当前 Store
//  2. 从本地对象映射表中查找对象
//  3. 编码对象并发送响应
//  4. 如果对象是 Stream，启动 goroutine 异步发送所有数据块
func (s *Actor) onObjectRequest(ctx actor.Context, req *cluster.ObjectRequest) {
	if req.Target != s.id {
		ctx.Logger().Warn("store: object request target not match",
			"id", req.ID,
			"target", req.Target,
			"store", s.id,
		)
		return
	}

	ctx.Logger().Info("store: responding object",
		"id", req.ID,
		"replyTo", req.ReplyTo,
	)

	reply := &cluster.ObjectResponse{ID: req.ID, Target: req.ReplyTo}
	obj, ok := s.localObjects[req.ID]
	if !ok {
		reply.Error = errors.Format("store: object %s not found", req.ID).Error()
	} else if encoded, err := obj.Encode(); err != nil {
		reply.Error = errors.WrapWith(err, "store: object %s encode failed", req.ID).Error()
	} else {
		reply.Value = encoded
	}
	s.router.Send(req.ReplyTo, reply)

	// if requested object is a stream, send all chunks
	if stream, ok := obj.(*object.Stream); ok {
		go func() {
			defer s.router.Send(req.ReplyTo, proto.NewStreamEnd(req.ID, req.ReplyTo))
			for obj := range stream.ToChan() {
				encoded, err := obj.Encode()
				msg := proto.NewStreamChunk(req.ID, req.ReplyTo, encoded, err)
				s.router.Send(req.ReplyTo, msg)
			}
		}()
	}
}

// onObjectResponse 处理来自远程 Store 的对象响应
// 参数:
//   - ctx: Actor 上下文
//   - resp: 对象响应消息
//
// 执行流程:
//  1. 检查响应是否有错误
//  2. 查找对应的 Future 对象
//  3. 如果是普通对象，直接 Resolve Future
//  4. 如果是 Stream，创建 Stream 对象并 Resolve Future
func (s *Actor) onObjectResponse(ctx actor.Context, resp *cluster.ObjectResponse) {
	if resp.Error != "" {
		ctx.Logger().Error("store: object response error",
			"id", resp.ID,
			"error", resp.Error,
		)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx.Logger().Info("store: receive object response",
		"id", resp.ID,
	)

	fut, ok := s.remoteObjects[resp.ID]
	if !ok {
		return
	}

	encoded := resp.Value
	if !encoded.Stream {
		fut.Resolve(encoded)
	} else {
		// ls := object.NewStream(nil, encoded.Language)
		ls := object.StreamWithID(resp.ID, nil, encoded.Language)
		fut.Resolve(ls)
	}
}

// onStreamChunk 处理来自远程 Store 的流数据块
// 参数:
//   - ctx: Actor 上下文
//   - chunk: 流数据块消息
//
// 执行流程:
//  1. 查找对应的 Future 对象
//  2. 从 Future 获取 Stream 对象
//  3. 如果是流结束标记（EoS），入队 nil 表示结束
//  4. 否则将数据块入队到 Stream
func (s *Actor) onStreamChunk(ctx actor.Context, chunk *proto.StreamChunk) {
	ctx.Logger().Info("store: receive stream chunk",
		"stream", chunk.StreamID,
		"isEos", chunk.EoS,
	)
	s.mu.RLock()
	defer s.mu.RUnlock()

	fut, ok := s.remoteObjects[chunk.StreamID]
	if !ok {
		return
	}

	obj, err := fut.Result()
	if err != nil {
		return
	}

	stream, ok := obj.(*object.Stream)
	if !ok {
		return
	}

	if chunk.EoS {
		stream.EnqueueChunk(nil)
	} else {
		stream.EnqueueChunk(chunk.Value)
	}
}

// onSaveObject 处理保存对象的请求
// 参数:
//   - ctx: Actor 上下文
//   - save: 保存对象消息
//
// 执行流程:
//  1. 将对象保存到本地对象映射表
//  2. 如果有回调函数，调用回调通知保存完成
func (s *Actor) onSaveObject(ctx actor.Context, save *SaveObject) {
	obj := save.Value
	ctx.Logger().Info("store: save object",
		"id", obj.GetID(),
	)

	s.localObjects[obj.GetID()] = obj

	if save.Callback != nil {
		save.Callback(ctx, &proto.Flow{ID: obj.GetID(), Source: s.ref})
	}
}

// getLocalObject 从本地对象映射表中获取对象
// 参数:
//   - flow: Flow 引用（包含对象 ID）
//
// 返回值:
//   - object.Interface: 对象实例
//   - error: 对象不存在的错误
func (s *Actor) getLocalObject(flow *proto.Flow) (object.Interface, error) {
	obj, ok := s.localObjects[flow.ID]
	if !ok {
		return nil, errors.Format("store: flow %s not found", flow.ID)
	}
	return obj, nil
}

// requestRemoteObject 请求远程 Store 中的对象
// 参数:
//   - ctx: Actor 上下文
//   - flow: Flow 引用（包含对象 ID 和来源 Store）
//
// 返回值:
//   - utils.Future[object.Interface]: Future 对象，用于异步获取结果
//
// 执行流程:
//  1. 检查是否已经发起过请求（避免重复请求）
//  2. 创建 Future 对象并注册
//  3. 通过路由器发送请求到远程 Store
//  4. 返回 Future 供调用者等待结果
func (s *Actor) requestRemoteObject(ctx actor.Context, flow *proto.Flow) utils.Future[object.Interface] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fut, ok := s.remoteObjects[flow.ID]; ok {
		return fut
	}

	fut := utils.NewFuture[object.Interface](configs.FlowTimeout)
	s.remoteObjects[flow.ID] = fut

	s.router.Send(flow.Source.ID, &cluster.ObjectRequest{
		ID:      flow.ID,
		Target:  flow.Source.ID,
		ReplyTo: s.id,
	})
	return fut
}

// onFlowRequest 处理本地 Actor 的对象请求
// 参数:
//   - ctx: Actor 上下文
//   - req: 请求对象消息
//
// 执行流程:
//  1. 判断对象是本地还是远程
//  2. 如果是本地对象，直接从本地映射表获取并回复
//  3. 如果是远程对象，请求远程 Store 并在 Future 完成时回复
func (s *Actor) onFlowRequest(ctx actor.Context, req *RequestObject) {
	ctx.Logger().Info("store: local flow request",
		"id", req.Flow.ID,
		"store", req.Flow.Source.ID,
	)

	if req.Flow.Source.ID == s.id {
		obj, err := s.getLocalObject(req.Flow)
		ctx.Send(req.ReplyTo, &ObjectResponse{
			Value: obj,
			Error: err,
		})
	} else {
		if s.cache != nil {
			if obj, ok := s.cache.Get(req.Flow.ID); ok {
				ctx.Logger().Info("store: loaded from cache", "obj", req.Flow.ID)
				ctx.Send(req.ReplyTo, &ObjectResponse{
					Value: obj,
				})
				return
			}
		}

		flow := req.Flow
		s.requestRemoteObject(ctx, flow).OnDone(func(obj object.Interface, duration time.Duration, err error) {
			ctx.Send(req.ReplyTo, &ObjectResponse{
				Value: obj,
				Error: err,
			})

			if _, ok := obj.(*object.Stream); ok {
				return
			}

			s.mu.RLock()
			defer s.mu.RUnlock()
			delete(s.remoteObjects, flow.ID)

			if s.cache == nil || err != nil {
				return
			}

			ctx.Logger().Info("store: insert remote cache", "object", flow.ID)
			s.cache.Insert(flow.ID, obj)
		})
	}
}

// onForward 处理需要转发的消息
// 参数:
//   - ctx: Actor 上下文
//   - forward: 转发消息
//
// 该方法通过路由器将消息转发到目标 Actor
func (s *Actor) onForward(ctx actor.Context, forward ForwardMessage) {
	ctx.Logger().Info("store: forwarding request",
		"target", forward.GetTarget(),
		"msg", forward,
	)
	s.router.Send(forward.GetTarget(), forward)
}

// Receive 实现 Actor 接口，处理接收到的消息
// 参数:
//   - ctx: Actor 上下文
//
// 支持的消息类型:
//   - cluster.ObjectRequest: 来自远程 Store 的对象请求
//   - cluster.ObjectResponse: 来自远程 Store 的对象响应
//   - cluster.StreamChunk: 流数据块
//   - SaveObject: 保存对象到本地
//   - RequestObject: 本地 Actor 请求对象
//   - ForwardMessage: 需要转发的消息
func (s *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	// Requests from remote stores
	case *cluster.ObjectRequest:
		s.onObjectRequest(ctx, msg)
	// Responses from remote stores
	case *cluster.ObjectResponse:
		s.onObjectResponse(ctx, msg)
	case *cluster.StreamChunk:
		s.onStreamChunk(ctx, msg)
	// Local actor requests
	case *SaveObject:
		s.onSaveObject(ctx, msg)
	case *RequestObject:
		s.onFlowRequest(ctx, msg)
	// forward messages
	case ForwardMessage:
		s.onForward(ctx, msg)
	default:
		ctx.Logger().Warn("store: unknown message type", "store", s.id, "type", msg)
	}
}

// New 创建一个新的 Store Actor 的 Props
// 参数:
//   - stub: 传输存根（用于远程通信）
//   - id: Store 标识符
//
// 返回值:
//   - *actor.Props: Actor 属性配置
//
// 该函数创建 Props 并在 Actor 初始化时设置 StoreRef
func New(router router.Router, id string) *actor.Props {
	s := &Actor{
		id:            id,
		router:        router,
		localObjects:  make(map[string]object.Interface),
		remoteObjects: make(map[string]utils.Future[object.Interface]),
		cache:         cache.NewLRU[string, object.Interface](32),
	}
	return actor.PropsFromProducer(func() actor.Actor {
		return s
	}, actor.WithOnInit(func(ctx actor.Context) {
		s.ref = &proto.StoreRef{ID: id, PID: ctx.Self()}
	}))
}

// Spawn 创建并启动一个 Store Actor
// 参数:
//   - ctx: Actor 根上下文
//   - stub: 传输存根（用于远程通信，可为 nil）
//   - id: Store 标识符
//
// 返回值:
//   - *proto.StoreRef: Store 引用（包含 ID 和 PID）
//
// 该函数会：
//  1. 创建 Store Actor 实例
//  2. 使用指定 ID 启动 Actor
//  3. 在路由器中注册 Store
//  4. 返回 Store 引用供其他组件使用
func Spawn(
	ctx *actor.RootContext,
	router router.Router,
	id string,
	cache ...cache.Controller[string, object.Interface],
) *proto.StoreRef {
	s := &Actor{
		id:            id,
		router:        router,
		localObjects:  make(map[string]object.Interface),
		remoteObjects: make(map[string]utils.Future[object.Interface]),
	}
	if len(cache) > 0 {
		s.cache = cache[0]
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return s
	})
	pid, _ := ctx.SpawnNamed(props, id)
	ref := &proto.StoreRef{
		ID:  id,
		PID: pid,
	}
	s.ref = ref

	router.Register(ref)

	return ref
}

// GetObject 从 Store 中获取对象（供同系统内的 Actor 调用）
// 参数:
//   - ctx: Actor 上下文
//   - store: Store Actor 的 PID
//   - flow: Flow 引用（包含对象 ID 和来源 Store）
//
// 返回值:
//   - utils.Future[object.Interface]: Future 对象，异步返回结果或错误
//
// 工作原理:
//  1. 创建一个临时的 Flow Actor 用于接收响应
//  2. 向 Store 发送 RequestObject 消息
//  3. Flow Actor 接收 ObjectResponse 并完成 Future
//  4. 响应后 Flow Actor 自动退出
//
// 该方法支持获取本地和远程的对象，Store 会自动处理路由
func GetObject(ctx actor.Context, store *actor.PID, flow *proto.Flow) utils.Future[object.Interface] {
	fut := utils.NewFuture[object.Interface](configs.FlowTimeout)
	if flow == nil {
		fut.Reject(errors.New("flow is nil"))
		return fut
	}

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *ObjectResponse:
			ctx.Logger().Info("store: flow response",
				"id", flow.ID,
				"value", msg.Value,
				"valueID", msg.Value.GetID(),
				"error", msg.Error,
			)

			if msg.Error != nil {
				fut.Reject(errors.WrapWith(msg.Error, "flow %s fetch failed", flow.ID))
				return
			}

			if msg.Value.GetID() != flow.ID {
				err := errors.Format("flow %s received unexpected ID %s", flow.ID, msg.Value.GetID())
				fut.Reject(err)
				return
			}

			fut.Resolve(msg.Value)
			c.Stop(c.Self()) // exit actor
		}
	})

	flowActor := ctx.Spawn(props)
	// if err != nil {
	// 	ctx.Logger().Error("store: flow spawn failed",
	// 		"id", flow.ID,
	// 		"error", err,
	// 	)
	// 	fut.Reject(errors.WrapWith(err, "flow %s spawn failed", flow.ID))
	// 	return fut
	// }
	ctx.Send(store, &RequestObject{
		ReplyTo: flowActor,
		Flow:    flow,
	})

	return fut
}
