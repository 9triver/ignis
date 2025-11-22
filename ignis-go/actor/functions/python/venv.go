// Package functions 提供了函数调用的封装和适配功能
// 该文件实现了 Python 虚拟环境的运行时管理，包括进程启动、通信和执行控制
package python

import (
	"context"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// VirtualEnv 表示一个 Python 虚拟环境实例
// 管理 Python 解释器进程的生命周期、包依赖、函数执行和结果返回
//
// 工作原理:
//   - 通过 transport.Executor 与 Python 进程建立双向通信通道
//   - 使用 Future 模式处理异步执行结果
//   - 支持流式数据传输（Stream）
//   - 每个执行请求通过唯一的 corrId（correlation ID）关联请求和响应
type VirtualEnv struct {
	mu      sync.Mutex                                // 互斥锁，保护并发访问
	ctx     context.Context                           // 上下文，用于生命周期管理
	handler transport.Executor                        // 远程执行器，负责与 Python 进程通信
	started bool                                      // 标记虚拟环境是否已启动
	futures map[string]utils.Future[object.Interface] // Future 映射表，key 为 corrId
	streams map[string]*object.Stream                 // Stream 映射表，key 为 corrId

	Name     string   `json:"name"`     // 虚拟环境名称
	Exec     string   `json:"exec"`     // Python 解释器可执行文件路径
	Packages []string `json:"packages"` // 已安装的包列表
}

// Interpreter 返回 Python 解释器可执行文件的路径
func (v *VirtualEnv) Interpreter() string {
	return v.Exec
}

// RunPip 创建一个 pip 命令用于安装或管理 Python 包
// 参数:
//   - args: pip 命令参数（如 "install", "package_name"）
//
// 返回值:
//   - *exec.Cmd: 配置好的命令对象
//   - context.CancelFunc: 取消函数，用于超时控制
//
// 该命令设置了 300 秒的超时时间，调用者需要在使用完毕后调用 cancel 函数
func (v *VirtualEnv) RunPip(args ...string) (*exec.Cmd, context.CancelFunc) {
	args = append([]string{"-m", "pip"}, args...)
	cmdCtx, cancel := context.WithTimeout(v.ctx, 300*time.Second)
	return exec.CommandContext(cmdCtx, v.Exec, args...), cancel
}

// AddPackages 向虚拟环境中添加并安装 Python 包
// 参数:
//   - p: 要安装的包名列表
//
// 返回值:
//   - error: 安装过程中的错误
//
// 行为说明:
//   - 自动跳过已安装的包，避免重复安装
//   - 使用 pip install 命令逐个安装包
//   - 安装成功后将包名添加到 Packages 列表中
//   - 该方法是线程安全的（使用互斥锁保护）
func (v *VirtualEnv) AddPackages(p ...string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	pkgSet := utils.MakeSetFromSlice(v.Packages)
	for _, pkg := range p {
		if pkgSet.Contains(pkg) {
			continue
		}

		if err := func() error {
			cmd, cancel := v.RunPip("install", pkg)
			defer cancel()

			if err := cmd.Run(); err != nil {
				return errors.WrapWith(err, "venv %s: failed installing package %s", v.Name, pkg)
			}
			return nil
		}(); err != nil {
			return err
		}

		v.Packages = append(v.Packages, pkg)
		pkgSet.Add(pkg)
	}
	return nil
}

// Execute 在 Python 虚拟环境中执行函数或方法
// 参数:
//   - name: 函数名或对象名
//   - method: 方法名（如果是对象方法调用），为空表示直接调用函数
//   - args: 参数 map，key 为参数名，value 为参数值对象
//
// 返回值:
//   - utils.Future[object.Interface]: Future 对象，用于异步获取执行结果
//
// 执行流程:
//  1. 创建 Future 对象用于接收结果
//  2. 将所有参数编码为可传输的格式
//  3. 生成唯一的 corrId（correlation ID）并关联 Future
//  4. 发送执行请求到 Python 进程
//  5. 如果参数中包含 Stream 对象，启动 goroutine 异步传输流数据
//  6. 返回 Future 对象供调用者等待结果
//
// 注意: 该方法是异步的，不会阻塞等待执行完成
func (v *VirtualEnv) Execute(name, method string, args map[string]object.Interface) utils.Future[object.Interface] {
	fut := utils.NewFuture[object.Interface](configs.ExecutionTimeout)
	encoded := make(map[string]*object.Remote)

	// 编码所有参数
	for param, obj := range args {
		enc, err := obj.Encode()
		if err != nil {
			fut.Reject(err)
			return fut
		}
		encoded[param] = enc
	}

	// 生成关联 ID 并注册 Future（加锁保护）
	v.mu.Lock()
	corrId := utils.GenID()
	v.futures[corrId] = fut
	v.mu.Unlock()

	// 注册 Future 完成回调，确保超时或完成后清理资源
	fut.OnDone(func(_ object.Interface, _ time.Duration, _ error) {
		// Future 已完成（无论成功、失败还是超时），确保清理 futures 映射
		v.mu.Lock()
		delete(v.futures, corrId)
		v.mu.Unlock()
	})

	// 发送执行请求
	msg := executor.NewExecute(v.Name, corrId, name, method, encoded)
	v.handler.SendChan() <- msg

	// 处理流式参数：为每个 Stream 参数启动独立的 goroutine 传输数据
	for _, arg := range args {
		if stream, ok := arg.(*object.Stream); ok {
			chunks := stream.ToChan()
			// 捕获 stream 变量，避免闭包问题
			go func(s *object.Stream, ch <-chan object.Interface) {
				defer func() {
					// 发送流结束标记，带 context 控制
					select {
					case v.handler.SendChan() <- executor.NewStreamEnd(v.Name, s.GetID()):
					case <-v.ctx.Done():
						return
					}
				}()

				// 持续传输流数据块，带 context 控制
				for {
					select {
					case chunk, ok := <-ch:
						if !ok {
							// channel 已关闭
							return
						}
						encoded, err := chunk.Encode()
						select {
						case v.handler.SendChan() <- executor.NewStreamChunk(v.Name, s.GetID(), encoded, err):
							// 发送成功，继续
						case <-v.ctx.Done():
							// context 已取消，退出
							return
						}
					case <-v.ctx.Done():
						// context 已取消，退出
						return
					}
				}
			}(stream, chunks)
		}
	}
	return fut
}

// Send 向 Python 执行器发送消息
// 参数:
//   - msg: 要发送的执行器消息
//
// 该方法用于向 Python 进程发送自定义消息，如添加函数处理器等
func (v *VirtualEnv) Send(msg *executor.Message) {
	v.handler.SendChan() <- msg
}

// onReturn 处理 Python 执行器返回的结果
// 参数:
//   - ret: 返回消息，包含 corrId 和执行结果
//
// 执行流程:
//  1. 通过 corrId 查找对应的 Future 对象
//  2. 从返回消息中解码结果对象
//  3. 如果返回的是 Stream，创建 Stream 对象并注册
//  4. 将结果传递给 Future，完成异步调用
//  5. 清理 futures 映射表中的条目
func (v *VirtualEnv) onReturn(ret *executor.Return) {
	v.mu.Lock()
	fut, ok := v.futures[ret.CorrID]
	if !ok {
		v.mu.Unlock()
		return
	}
	delete(v.futures, ret.CorrID)
	v.mu.Unlock()

	obj, err := ret.Object()
	if err != nil {
		fut.Reject(err)
		return
	}

	var o object.Interface
	if obj.Stream { // 返回值是 Stream
		values := make(chan object.Interface)
		ls := object.NewStream(values, obj.GetLanguage())

		v.mu.Lock()
		v.streams[ret.CorrID] = ls
		v.mu.Unlock()

		o = ls
	} else {
		o = obj
	}
	fut.Resolve(o)
}

// onStreamChunk 处理从 Python 返回的流数据块
// 参数:
//   - chunk: 流数据块消息
//
// 执行流程:
//  1. 通过 StreamID 查找对应的 Stream 对象
//  2. 如果是流结束标记（EoS），将 nil 入队表示流结束，并清理映射
//  3. 否则将数据块入队到 Stream 中
func (v *VirtualEnv) onStreamChunk(chunk *proto.StreamChunk) {
	v.mu.Lock()
	stream, ok := v.streams[chunk.StreamID]
	v.mu.Unlock()

	if !ok {
		return
	}

	if chunk.EoS {
		stream.EnqueueChunk(nil)
		// 清理 streams 映射，防止内存泄漏
		v.mu.Lock()
		delete(v.streams, chunk.StreamID)
		v.mu.Unlock()
	} else {
		stream.EnqueueChunk(chunk.GetValue())
	}
}

// onReceive 接收并分发来自 Python 执行器的消息
// 参数:
//   - msg: 执行器消息
//
// 根据消息类型分发到不同的处理器:
//   - StreamChunk: 流数据块 -> onStreamChunk
//   - Return: 执行结果 -> onReturn
func (v *VirtualEnv) onReceive(msg *executor.Message) {
	switch cmd := msg.Command.(type) {
	case *executor.Message_StreamChunk:
		v.onStreamChunk(cmd.StreamChunk)
	case *executor.Message_Return:
		v.onReturn(cmd.Return)
	}
}

// Run 启动 Python 虚拟环境并建立通信通道
// 参数:
//   - addr: 远程执行器的地址，用于 Python 进程连接
//
// 执行流程:
//  1. 检查虚拟环境是否已启动，避免重复启动
//  2. 启动 Python 解释器进程，运行执行器脚本 (__actor_executor.py)
//     - 传递 --remote 参数指定连接地址
//     - 传递 --venv 参数指定虚拟环境名称
//  3. 将 Python 进程的标准输出和标准错误重定向到当前进程
//  4. 启动消息接收 goroutine，持续监听并处理来自 Python 的消息
//  5. 等待执行器准备就绪（通过 handler.Ready() 信号）
//
// 注意:
//   - 该方法是线程安全的（使用互斥锁保护）
//   - Python 进程和消息接收循环都在独立的 goroutine 中运行
//   - 方法会阻塞直到执行器准备就绪
func (v *VirtualEnv) Run(addr string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.started {
		return
	}
	v.started = true

	// 启动 Python 执行器进程
	go func() {
		basePath, err := getVenvPath()
		if err != nil {
			// 无法获取虚拟环境路径，记录错误并返回
			// TODO: 考虑添加错误通知机制
			return
		}
		cmd := exec.CommandContext(v.ctx, v.Exec, path.Join(basePath, v.Name, venvStart), "--remote", addr, "--venv", v.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Python 进程执行失败
			// TODO: 考虑添加错误通知机制
			return
		}
	}()

	// 启动消息接收循环
	go func() {
		for msg := range v.handler.RecvChan() {
			v.onReceive(msg)
		}
	}()

	// 等待执行器准备就绪
	<-v.handler.Ready()
}
