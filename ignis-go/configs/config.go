package configs

import "context"

// ResourceInfo 表示资源信息
type ResourceInfo struct {
	CPU    int64 `json:"cpu"`    // CPU核心数
	Memory int64 `json:"memory"` // 内存(字节)
	GPU    int64 `json:"gpu"`    // GPU数量
}

// ResourceCapacity 表示资源容量信息
type ResourceCapacity struct {
	Total     ResourceInfo `json:"total"`     // 总资源
	Used      ResourceInfo `json:"used"`      // 已使用资源
	Available ResourceInfo `json:"available"` // 可用资源
}

type ResourceProviderMap map[string]*ResourceCapacity

// ResourceProviderFunc 定义获取可用资源的函数类型
// 返回provider ID到资源容量信息的映射
type GetResourceProviders func(ctx context.Context) (ResourceProviderMap, error)

type Config struct {
	RpcAddr              string
	GetResourceProviders GetResourceProviders
}
