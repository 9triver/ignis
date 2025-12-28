// Package object 提供了流式对象的实现
package object

import (
	"reflect"
	"sync"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/utils"
)

// Stream 表示流式对象
// 支持多消费者模式，可以将数据分发给多个订阅者
//
// 工作原理:
//   - values channel 是数据源，生产者通过 EnqueueChunk 入队数据
//   - consumers 是消费者列表，每个消费者有独立的 channel
//   - 数据从 values 读取后广播到所有 consumers
//   - 使用 sync.Once 确保只启动一次分发循环
//   - 支持在流完成后动态添加消费者（会立即关闭）
type Stream struct {
	once      sync.Once        // 确保只启动一次分发循环
	mu        sync.RWMutex     // 保护 consumers 列表
	consumers []chan Interface // 消费者 channel 列表

	id        string         // 流的唯一标识符
	completed bool           // 标记流是否已完成
	language  Language       // 流数据所属的语言类型
	values    chan Interface // 数据源 channel
}

// EnqueueChunk 将数据块入队到流中
// 参数:
//   - chunk: 数据块（Interface 对象），nil 表示流结束
//
// 行为:
//   - 如果流已完成，忽略新数据
//   - 如果 chunk 为 nil，标记流完成并关闭 values channel
//   - 否则将数据块发送到 values channel
func (s *Stream) EnqueueChunk(chunk Interface) {
	if s.completed {
		return
	}

	if chunk == nil {
		s.completed = true
		close(s.values)
		return
	}

	s.values <- chunk
}

// GetID 返回流的唯一标识符
func (s *Stream) GetID() string {
	return s.id
}

// GetLanguage 返回流数据所属的语言类型
func (s *Stream) GetLanguage() Language {
	return s.language
}

// Encode 将流对象编码为可传输的 Remote 格式
// 返回值:
//   - *Remote: 编码后的远程对象（标记为 Stream）
//   - error: 编码错误（Stream 不会出错）
//
// 注意: 流的实际数据不会被编码，只传输元数据
// 数据通过 StreamChunk 消息逐块传输
func (s *Stream) Encode() (*Remote, error) {
	return &Remote{
		ID:       s.id,
		Language: s.language,
		Stream:   true,
	}, nil
}

// Value 返回流的数据源 channel
// 返回值:
//   - any: values channel
//   - error: 获取错误（Stream 不会出错）
func (s *Stream) Value() (any, error) {
	return s.values, nil
}

// doStart 启动数据分发循环
// 该方法在后台运行，从 values channel 读取数据并广播到所有 consumers
//
// 执行流程:
//  1. 持续从 values channel 读取数据块
//  2. 将每个数据块广播到所有消费者 channel
//  3. 当 values channel 关闭时，关闭所有消费者 channel
func (s *Stream) doStart() {
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		for _, ch := range s.consumers {
			close(ch)
		}
	}()

	for obj := range s.values {
		s.mu.RLock()
		for _, ch := range s.consumers {
			ch <- obj
		}
		s.mu.RUnlock()
	}
}

// ToChan 创建一个新的消费者 channel 并返回
// 返回值:
//   - <-chan Interface: 只读的消费者 channel
//
// 行为:
//   - 每次调用都会创建一个新的消费者 channel
//   - 第一次调用时启动分发循环（通过 sync.Once 保证）
//   - 所有消费者会同时接收到相同的数据
//   - 当流结束时，所有消费者 channel 会被关闭
//
// 典型用法:
//
//	ch := stream.ToChan()
//	for obj := range ch {
//	    // 处理对象
//	}
func (s *Stream) ToChan() <-chan Interface {
	ch := make(chan Interface)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consumers = append(s.consumers, ch)
	s.once.Do(func() {
		go s.doStart()
	})
	return ch
}

func (s *Stream) IsStream() bool {
	return true
}

func (s *Stream) Completed() bool {
	return s.completed
}

// NewStream 创建一个新的流对象，自动生成唯一 ID
// 参数:
//   - values: 数据源（可以是 channel 或 nil）
//   - language: 语言类型
//
// 返回值:
//   - *Stream: 流对象实例
//
// 如果 values 为 nil，返回空流（需要手动 EnqueueChunk）
// 如果 values 是 channel，会启动 goroutine 自动从 channel 读取数据
//
// ID 格式: "stream." + 随机字符串
func NewStream(values any, language Language) *Stream {
	return StreamWithID(utils.GenIDWith("stream."), values, language)
}

// StreamWithID 创建一个带指定 ID 的流对象
// 参数:
//   - id: 流的唯一标识符
//   - values: 数据源（可以是 channel 或 nil）
//   - language: 语言类型
//
// 返回值:
//   - *Stream: 流对象实例
//
// 如果 values 不为 nil，会使用反射从 channel 读取数据并入队
// 数据会自动包装为 Local 对象（如果不是 Interface 类型）
func StreamWithID(id string, values any, language Language) *Stream {
	s := &Stream{
		id:       id,
		values:   make(chan Interface, configs.ChannelBufferSize),
		language: language,
	}

	if values == nil {
		return s
	}

	// 启动 goroutine 从源 channel 读取数据
	go func() {
		defer s.EnqueueChunk(nil) // 流结束时发送结束信号

		ch := reflect.ValueOf(values)
		for {
			v, ok := ch.Recv()
			if !ok {
				// 源 channel 已关闭
				return
			}

			// 检查是否已经是 Interface 类型
			if obj, ok := v.Interface().(Interface); ok {
				s.EnqueueChunk(obj)
			} else {
				// 自动包装为 Local 对象
				s.EnqueueChunk(NewLocal(v.Interface(), language))
			}
		}
	}()

	return s
}
