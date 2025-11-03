package monitor

import (
	"context"
	"time"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
)

// Monitor 是状态监控的核心接口，用于替换 platform 中的 ApplicationInfo 功能
// 支持多种实现，可在创建 platform 时进行依赖注入
type Monitor interface {
	// ApplicationMonitor 应用级监控接口
	ApplicationMonitor
	// NodeMonitor 节点级监控接口
	NodeMonitor
	// TaskMonitor 任务级监控接口
	TaskMonitor
	// ResourceMonitor 资源监控接口
	ResourceMonitor
	// MetricsMonitor 指标监控接口
	MetricsMonitor
	// EventMonitor 事件监控接口
	EventMonitor
	// LifecycleManager 生命周期管理接口
	LifecycleManager
	// ObservableMonitor 可观察接口
	ObservableMonitor
}

// =============================================================================
// 应用级监控接口
// =============================================================================

// ApplicationMonitor 提供应用级别的监控功能
type ApplicationMonitor interface {
	// RegisterApplication 注册一个新应用
	RegisterApplication(ctx context.Context, appID string, metadata *ApplicationMetadata) error

	// UnregisterApplication 注销一个应用
	UnregisterApplication(ctx context.Context, appID string) error

	// GetApplicationInfo 获取应用的完整信息
	GetApplicationInfo(ctx context.Context, appID string) (*ApplicationInfo, error)

	// ListApplications 列出所有应用
	ListApplications(ctx context.Context, filter *ApplicationFilter) ([]*ApplicationInfo, error)

	// UpdateApplicationStatus 更新应用状态
	UpdateApplicationStatus(ctx context.Context, appID string, status ApplicationStatus) error

	// GetApplicationDAG 获取应用的DAG
	GetApplicationDAG(ctx context.Context, appID string) (*controller.DAG, error)

	// SetApplicationDAG 设置应用的DAG
	SetApplicationDAG(ctx context.Context, appID string, dag *controller.DAG) error

	// GetApplicationMetrics 获取应用级别的聚合指标
	GetApplicationMetrics(ctx context.Context, appID string) (*ApplicationMetrics, error)
}

// ApplicationMetadata 应用元数据
type ApplicationMetadata struct {
	AppID       string            `json:"appId"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Owner       string            `json:"owner"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// ApplicationStatus 应用状态
type ApplicationStatus string

const (
	AppStatusPending    ApplicationStatus = "pending"
	AppStatusRunning    ApplicationStatus = "running"
	AppStatusCompleted  ApplicationStatus = "completed"
	AppStatusFailed     ApplicationStatus = "failed"
	AppStatusCancelled  ApplicationStatus = "cancelled"
	AppStatusSuspended  ApplicationStatus = "suspended"
	AppStatusTerminated ApplicationStatus = "terminated"
)

// ApplicationInfo 应用完整信息
type ApplicationInfo struct {
	Metadata      *ApplicationMetadata `json:"metadata"`
	Status        ApplicationStatus    `json:"status"`
	DAG           *controller.DAG      `json:"dag"`
	Progress      *ApplicationProgress `json:"progress"`
	Resources     *ResourceUsage       `json:"resources"`
	StartTime     *time.Time           `json:"startTime,omitempty"`
	EndTime       *time.Time           `json:"endTime,omitempty"`
	Duration      time.Duration        `json:"duration"`
	ErrorMessage  string               `json:"errorMessage,omitempty"`
	ControllerRef *proto.ActorRef      `json:"controllerRef,omitempty"`
}

// ApplicationProgress 应用进度
type ApplicationProgress struct {
	TotalNodes     int     `json:"totalNodes"`
	CompletedNodes int     `json:"completedNodes"`
	FailedNodes    int     `json:"failedNodes"`
	RunningNodes   int     `json:"runningNodes"`
	PendingNodes   int     `json:"pendingNodes"`
	Percentage     float64 `json:"percentage"`
}

// ApplicationFilter 应用过滤器
type ApplicationFilter struct {
	Status    []ApplicationStatus `json:"status,omitempty"`
	Owner     string              `json:"owner,omitempty"`
	Tags      map[string]string   `json:"tags,omitempty"`
	CreatedAt *TimeRange          `json:"createdAt,omitempty"`
	Limit     int                 `json:"limit,omitempty"`
	Offset    int                 `json:"offset,omitempty"`
	SortBy    string              `json:"sortBy,omitempty"`    // "createdAt", "updatedAt", "name"
	SortOrder string              `json:"sortOrder,omitempty"` // "asc", "desc"
}

// ApplicationMetrics 应用级别的聚合指标
type ApplicationMetrics struct {
	AppID                  string        `json:"appId"`
	TotalExecutionTime     time.Duration `json:"totalExecutionTime"`
	AverageNodeLatency     time.Duration `json:"averageNodeLatency"`
	TotalDataProcessed     int64         `json:"totalDataProcessed"` // bytes
	TotalMessagesExchanged int64         `json:"totalMessagesExchanged"`
	ErrorRate              float64       `json:"errorRate"`
	Throughput             float64       `json:"throughput"` // tasks/second
}

// =============================================================================
// 节点级监控接口
// =============================================================================

// NodeMonitor 提供DAG节点级别的监控功能
type NodeMonitor interface {
	// GetNodeState 获取单个节点的状态
	GetNodeState(ctx context.Context, appID, nodeID string) (*NodeState, error)

	// GetAllNodeStates 获取应用所有节点的状态
	GetAllNodeStates(ctx context.Context, appID string) (map[string]*NodeState, error)

	// UpdateNodeState 更新节点状态
	UpdateNodeState(ctx context.Context, appID, nodeID string, state *NodeState) error

	// MarkNodeReady 标记节点为就绪状态
	MarkNodeReady(ctx context.Context, appID, nodeID string) error

	// MarkNodeRunning 标记节点为运行中状态
	MarkNodeRunning(ctx context.Context, appID, nodeID string) error

	// MarkNodeDone 标记节点为完成状态
	MarkNodeDone(ctx context.Context, appID, nodeID string, result *NodeResult) error

	// MarkNodeFailed 标记节点为失败状态
	MarkNodeFailed(ctx context.Context, appID, nodeID string, err error) error

	// GetNodeMetrics 获取节点的执行指标
	GetNodeMetrics(ctx context.Context, appID, nodeID string) (*NodeMetrics, error)

	// GetNodeDependencies 获取节点的依赖关系
	GetNodeDependencies(ctx context.Context, appID, nodeID string) (*NodeDependencies, error)

	// WatchNodeState 监听节点状态变化
	WatchNodeState(ctx context.Context, appID, nodeID string) (<-chan *NodeStateChangeEvent, error)
}

// NodeStatus 节点状态
type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusReady     NodeStatus = "ready"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
	NodeStatusSkipped   NodeStatus = "skipped"
	NodeStatusCancelled NodeStatus = "cancelled"
)

// NodeType 节点类型
type NodeType string

const (
	NodeTypeControl NodeType = "control"
	NodeTypeData    NodeType = "data"
	NodeTypeCompute NodeType = "compute"
)

// NodeState 节点状态信息
type NodeState struct {
	ID           string            `json:"id"`
	Type         NodeType          `json:"type"`
	Status       NodeStatus        `json:"status"`
	StartTime    *time.Time        `json:"startTime,omitempty"`
	EndTime      *time.Time        `json:"endTime,omitempty"`
	Duration     time.Duration     `json:"duration"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	RetryCount   int               `json:"retryCount"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	Result       *NodeResult       `json:"result,omitempty"`
	ActorRef     *proto.ActorRef   `json:"actorRef,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// NodeResult 节点执行结果
type NodeResult struct {
	Success      bool          `json:"success"`
	OutputData   []*proto.Flow `json:"outputData,omitempty"`
	ErrorDetails *ErrorDetails `json:"errorDetails,omitempty"`
	Metrics      *NodeMetrics  `json:"metrics,omitempty"`
}

// ErrorDetails 错误详情
type ErrorDetails struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	StackTrace  string    `json:"stackTrace,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// NodeMetrics 节点执行指标
type NodeMetrics struct {
	NodeID          string        `json:"nodeId"`
	ExecutionTime   time.Duration `json:"executionTime"`
	QueueTime       time.Duration `json:"queueTime"`
	CalcLatency     time.Duration `json:"calcLatency"`
	LinkLatency     time.Duration `json:"linkLatency"`
	InputDataSize   int64         `json:"inputDataSize"`  // bytes
	OutputDataSize  int64         `json:"outputDataSize"` // bytes
	MemoryUsage     int64         `json:"memoryUsage"`    // bytes
	CPUUsage        float64       `json:"cpuUsage"`       // percentage
	NetworkBytesIn  int64         `json:"networkBytesIn"`
	NetworkBytesOut int64         `json:"networkBytesOut"`
}

// NodeDependencies 节点依赖关系
type NodeDependencies struct {
	NodeID       string   `json:"nodeId"`
	Predecessors []string `json:"predecessors"` // 前驱节点
	Successors   []string `json:"successors"`   // 后继节点
	DataDeps     []string `json:"dataDeps"`     // 数据依赖
	ControlDeps  []string `json:"controlDeps"`  // 控制依赖
}

// NodeStateChangeEvent 节点状态变化事件
type NodeStateChangeEvent struct {
	AppID     string     `json:"appId"`
	NodeID    string     `json:"nodeId"`
	OldStatus NodeStatus `json:"oldStatus"`
	NewStatus NodeStatus `json:"newStatus"`
	State     *NodeState `json:"state"`
	Timestamp time.Time  `json:"timestamp"`
}

// =============================================================================
// 任务级监控接口
// =============================================================================

// TaskMonitor 提供任务级别的监控功能
type TaskMonitor interface {
	// RecordTaskStart 记录任务开始
	RecordTaskStart(ctx context.Context, taskID string, task *TaskInfo) error

	// RecordTaskEnd 记录任务结束
	RecordTaskEnd(ctx context.Context, taskID string, result *TaskResult) error

	// GetTaskInfo 获取任务信息
	GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error)

	// ListTasks 列出任务
	ListTasks(ctx context.Context, appID string, filter *TaskFilter) ([]*TaskInfo, error)

	// GetTaskMetrics 获取任务指标
	GetTaskMetrics(ctx context.Context, taskID string) (*TaskMetrics, error)

	// CancelTask 取消任务
	CancelTask(ctx context.Context, taskID string) error
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID       string            `json:"taskId"`
	AppID        string            `json:"appId"`
	NodeID       string            `json:"nodeId"`
	Status       TaskStatus        `json:"status"`
	FunctionName string            `json:"functionName"`
	Parameters   map[string]string `json:"parameters,omitempty"`
	StartTime    *time.Time        `json:"startTime,omitempty"`
	EndTime      *time.Time        `json:"endTime,omitempty"`
	Duration     time.Duration     `json:"duration"`
	WorkerID     string            `json:"workerId,omitempty"`
	RetryCount   int               `json:"retryCount"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID       string        `json:"taskId"`
	Success      bool          `json:"success"`
	ResultData   interface{}   `json:"resultData,omitempty"`
	ErrorDetails *ErrorDetails `json:"errorDetails,omitempty"`
	Metrics      *TaskMetrics  `json:"metrics,omitempty"`
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	Status    []TaskStatus `json:"status,omitempty"`
	NodeID    string       `json:"nodeId,omitempty"`
	WorkerID  string       `json:"workerId,omitempty"`
	StartTime *TimeRange   `json:"startTime,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	TaskID        string        `json:"taskId"`
	ExecutionTime time.Duration `json:"executionTime"`
	WaitTime      time.Duration `json:"waitTime"`
	DataSize      int64         `json:"dataSize"`
	MemoryPeak    int64         `json:"memoryPeak"`
	CPUTime       time.Duration `json:"cpuTime"`
}

// =============================================================================
// 资源监控接口
// =============================================================================

// ResourceMonitor 提供资源监控功能
type ResourceMonitor interface {
	// RecordResourceUsage 记录资源使用情况
	RecordResourceUsage(ctx context.Context, appID string, usage *ResourceUsage) error

	// GetResourceUsage 获取当前资源使用情况
	GetResourceUsage(ctx context.Context, appID string) (*ResourceUsage, error)

	// GetResourceHistory 获取资源使用历史
	GetResourceHistory(ctx context.Context, appID string, timeRange *TimeRange) ([]*ResourceSnapshot, error)

	// GetWorkerResources 获取Worker资源信息
	GetWorkerResources(ctx context.Context, workerID string) (*WorkerResources, error)

	// ListWorkers 列出所有Worker
	ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*WorkerInfo, error)

	// GetClusterResources 获取集群资源汇总
	GetClusterResources(ctx context.Context) (*ClusterResources, error)
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	AppID           string    `json:"appId"`
	CPUUsage        float64   `json:"cpuUsage"`        // percentage
	MemoryUsed      int64     `json:"memoryUsed"`      // bytes
	MemoryTotal     int64     `json:"memoryTotal"`     // bytes
	DiskUsed        int64     `json:"diskUsed"`        // bytes
	DiskTotal       int64     `json:"diskTotal"`       // bytes
	NetworkBytesIn  int64     `json:"networkBytesIn"`  // bytes
	NetworkBytesOut int64     `json:"networkBytesOut"` // bytes
	ActiveWorkers   int       `json:"activeWorkers"`
	TotalWorkers    int       `json:"totalWorkers"`
	ActiveActors    int       `json:"activeActors"`
	Timestamp       time.Time `json:"timestamp"`
}

// ResourceSnapshot 资源使用快照
type ResourceSnapshot struct {
	Usage     *ResourceUsage `json:"usage"`
	Timestamp time.Time      `json:"timestamp"`
}

// WorkerResources Worker资源信息
type WorkerResources struct {
	WorkerID    string    `json:"workerId"`
	CPUCores    int       `json:"cpuCores"`
	CPUUsage    float64   `json:"cpuUsage"`
	MemoryTotal int64     `json:"memoryTotal"`
	MemoryUsed  int64     `json:"memoryUsed"`
	DiskTotal   int64     `json:"diskTotal"`
	DiskUsed    int64     `json:"diskUsed"`
	NetworkIn   int64     `json:"networkIn"`
	NetworkOut  int64     `json:"networkOut"`
	ActiveTasks int       `json:"activeTasks"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WorkerInfo Worker信息
type WorkerInfo struct {
	WorkerID     string            `json:"workerId"`
	HostName     string            `json:"hostName"`
	IPAddress    string            `json:"ipAddress"`
	Status       WorkerStatus      `json:"status"`
	Resources    *WorkerResources  `json:"resources"`
	Capabilities []string          `json:"capabilities"`
	Labels       map[string]string `json:"labels"`
	StartTime    time.Time         `json:"startTime"`
	LastSeen     time.Time         `json:"lastSeen"`
}

// WorkerStatus Worker状态
type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"
	WorkerStatusOffline WorkerStatus = "offline"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusIdle    WorkerStatus = "idle"
)

// WorkerFilter Worker过滤器
type WorkerFilter struct {
	Status       []WorkerStatus    `json:"status,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	Offset       int               `json:"offset,omitempty"`
}

// ClusterResources 集群资源汇总
type ClusterResources struct {
	TotalCPUCores       int       `json:"totalCpuCores"`
	UsedCPUCores        float64   `json:"usedCpuCores"`
	TotalMemory         int64     `json:"totalMemory"`
	UsedMemory          int64     `json:"usedMemory"`
	TotalDisk           int64     `json:"totalDisk"`
	UsedDisk            int64     `json:"usedDisk"`
	TotalWorkers        int       `json:"totalWorkers"`
	OnlineWorkers       int       `json:"onlineWorkers"`
	TotalApplications   int       `json:"totalApplications"`
	RunningApplications int       `json:"runningApplications"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// =============================================================================
// 指标监控接口
// =============================================================================

// MetricsMonitor 提供指标收集和查询功能
type MetricsMonitor interface {
	// RecordMetric 记录单个指标
	RecordMetric(ctx context.Context, metric *Metric) error

	// RecordMetrics 批量记录指标
	RecordMetrics(ctx context.Context, metrics []*Metric) error

	// QueryMetrics 查询指标
	QueryMetrics(ctx context.Context, query *MetricQuery) ([]*Metric, error)

	// GetMetricAggregation 获取指标聚合结果
	GetMetricAggregation(ctx context.Context, query *MetricQuery, aggregation *AggregationConfig) (*AggregationResult, error)

	// ListMetricNames 列出所有指标名称
	ListMetricNames(ctx context.Context, prefix string) ([]string, error)

	// DeleteMetrics 删除指标数据
	DeleteMetrics(ctx context.Context, query *MetricQuery) error
}

// MetricType 指标类型
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"   // 计数器
	MetricTypeGauge     MetricType = "gauge"     // 仪表盘
	MetricTypeHistogram MetricType = "histogram" // 直方图
	MetricTypeSummary   MetricType = "summary"   // 摘要
)

// Metric 指标
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	Unit      string            `json:"unit,omitempty"`
}

// MetricQuery 指标查询
type MetricQuery struct {
	Names     []string          `json:"names,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	TimeRange *TimeRange        `json:"timeRange,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
}

// AggregationConfig 聚合配置
type AggregationConfig struct {
	Type     AggregationType `json:"type"`
	Interval time.Duration   `json:"interval,omitempty"` // 时间间隔聚合
	GroupBy  []string        `json:"groupBy,omitempty"`  // 按标签分组
}

// AggregationType 聚合类型
type AggregationType string

const (
	AggregationTypeSum   AggregationType = "sum"
	AggregationTypeAvg   AggregationType = "avg"
	AggregationTypeMin   AggregationType = "min"
	AggregationTypeMax   AggregationType = "max"
	AggregationTypeCount AggregationType = "count"
	AggregationTypeP50   AggregationType = "p50"
	AggregationTypeP95   AggregationType = "p95"
	AggregationTypeP99   AggregationType = "p99"
)

// AggregationResult 聚合结果
type AggregationResult struct {
	Query       *MetricQuery            `json:"query"`
	Aggregation *AggregationConfig      `json:"aggregation"`
	Results     []*AggregationDataPoint `json:"results"`
}

// AggregationDataPoint 聚合数据点
type AggregationDataPoint struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Count     int64             `json:"count,omitempty"`
}

// =============================================================================
// 事件监控接口
// =============================================================================

// EventMonitor 提供事件记录和查询功能
type EventMonitor interface {
	// RecordEvent 记录事件
	RecordEvent(ctx context.Context, event *Event) error

	// QueryEvents 查询事件
	QueryEvents(ctx context.Context, query *EventQuery) ([]*Event, error)

	// Subscribe 订阅事件
	Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error)

	// Unsubscribe 取消订阅
	Unsubscribe(ctx context.Context, subscriptionID string) error

	// GetEventStatistics 获取事件统计
	GetEventStatistics(ctx context.Context, query *EventQuery) (*EventStatistics, error)
}

// EventType 事件类型
type EventType string

const (
	EventTypeApplicationRegistered   EventType = "application.registered"
	EventTypeApplicationUnregistered EventType = "application.unregistered"
	EventTypeApplicationStarted      EventType = "application.started"
	EventTypeApplicationCompleted    EventType = "application.completed"
	EventTypeApplicationFailed       EventType = "application.failed"
	EventTypeNodeStateChanged        EventType = "node.state_changed"
	EventTypeNodeStarted             EventType = "node.started"
	EventTypeNodeCompleted           EventType = "node.completed"
	EventTypeNodeFailed              EventType = "node.failed"
	EventTypeTaskStarted             EventType = "task.started"
	EventTypeTaskCompleted           EventType = "task.completed"
	EventTypeTaskFailed              EventType = "task.failed"
	EventTypeWorkerJoined            EventType = "worker.joined"
	EventTypeWorkerLeft              EventType = "worker.left"
	EventTypeResourceAlert           EventType = "resource.alert"
	EventTypeError                   EventType = "error"
	EventTypeWarning                 EventType = "warning"
	EventTypeInfo                    EventType = "info"
)

// EventLevel 事件级别
type EventLevel string

const (
	EventLevelDebug    EventLevel = "debug"
	EventLevelInfo     EventLevel = "info"
	EventLevelWarning  EventLevel = "warning"
	EventLevelError    EventLevel = "error"
	EventLevelCritical EventLevel = "critical"
)

// Event 事件
type Event struct {
	ID        string            `json:"id"`
	Type      EventType         `json:"type"`
	Level     EventLevel        `json:"level"`
	AppID     string            `json:"appId,omitempty"`
	NodeID    string            `json:"nodeId,omitempty"`
	TaskID    string            `json:"taskId,omitempty"`
	WorkerID  string            `json:"workerId,omitempty"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Source    string            `json:"source,omitempty"`
}

// EventQuery 事件查询
type EventQuery struct {
	Types     []EventType  `json:"types,omitempty"`
	Levels    []EventLevel `json:"levels,omitempty"`
	AppID     string       `json:"appId,omitempty"`
	NodeID    string       `json:"nodeId,omitempty"`
	WorkerID  string       `json:"workerId,omitempty"`
	TimeRange *TimeRange   `json:"timeRange,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
	SortOrder string       `json:"sortOrder,omitempty"` // "asc", "desc"
}

// EventFilter 事件过滤器
type EventFilter struct {
	Types  []EventType  `json:"types,omitempty"`
	Levels []EventLevel `json:"levels,omitempty"`
	AppID  string       `json:"appId,omitempty"`
	NodeID string       `json:"nodeId,omitempty"`
}

// EventStatistics 事件统计
type EventStatistics struct {
	TotalEvents     int64                `json:"totalEvents"`
	EventsByType    map[EventType]int64  `json:"eventsByType"`
	EventsByLevel   map[EventLevel]int64 `json:"eventsByLevel"`
	TimeRange       *TimeRange           `json:"timeRange"`
	EventsPerMinute float64              `json:"eventsPerMinute"`
}

// =============================================================================
// 生命周期管理接口
// =============================================================================

// LifecycleManager 提供监控系统的生命周期管理
type LifecycleManager interface {
	// Start 启动监控系统
	Start(ctx context.Context) error

	// Stop 停止监控系统
	Stop(ctx context.Context) error

	// Status 获取监控系统状态
	Status(ctx context.Context) (*MonitorStatus, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) (*HealthCheckResult, error)

	// Reset 重置监控系统（清空所有数据）
	Reset(ctx context.Context) error

	// Export 导出监控数据
	Export(ctx context.Context, config *ExportConfig) ([]byte, error)

	// Import 导入监控数据
	Import(ctx context.Context, data []byte) error
}

// MonitorStatus 监控系统状态
type MonitorStatus struct {
	Running           bool          `json:"running"`
	StartTime         time.Time     `json:"startTime"`
	Uptime            time.Duration `json:"uptime"`
	TotalApplications int           `json:"totalApplications"`
	TotalEvents       int64         `json:"totalEvents"`
	TotalMetrics      int64         `json:"totalMetrics"`
	StorageUsed       int64         `json:"storageUsed"` // bytes
	MemoryUsed        int64         `json:"memoryUsed"`  // bytes
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Healthy       bool            `json:"healthy"`
	Checks        map[string]bool `json:"checks"`
	ErrorMessages []string        `json:"errorMessages,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// ExportConfig 导出配置
type ExportConfig struct {
	Format         ExportFormat `json:"format"`
	Applications   []string     `json:"applications,omitempty"` // 空表示全部
	TimeRange      *TimeRange   `json:"timeRange,omitempty"`
	IncludeMetrics bool         `json:"includeMetrics"`
	IncludeEvents  bool         `json:"includeEvents"`
	Compression    bool         `json:"compression"`
}

// ExportFormat 导出格式
type ExportFormat string

const (
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatCSV      ExportFormat = "csv"
	ExportFormatProtobuf ExportFormat = "protobuf"
)

// =============================================================================
// 通用类型
// =============================================================================

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// =============================================================================
// 观察者模式支持
// =============================================================================

// Observer 观察者接口
type Observer interface {
	// OnApplicationStateChanged 应用状态变化回调
	OnApplicationStateChanged(event *ApplicationStateChangeEvent)

	// OnNodeStateChanged 节点状态变化回调
	OnNodeStateChanged(event *NodeStateChangeEvent)

	// OnEvent 事件回调
	OnEvent(event *Event)
}

// ApplicationStateChangeEvent 应用状态变化事件
type ApplicationStateChangeEvent struct {
	AppID     string            `json:"appId"`
	OldStatus ApplicationStatus `json:"oldStatus"`
	NewStatus ApplicationStatus `json:"newStatus"`
	Info      *ApplicationInfo  `json:"info"`
	Timestamp time.Time         `json:"timestamp"`
}

// ObservableMonitor 可观察的监控接口
type ObservableMonitor interface {
	// AddObserver 添加观察者
	AddObserver(observer Observer)

	// RemoveObserver 移除观察者
	RemoveObserver(observer Observer)

	// NotifyObservers 通知所有观察者
	NotifyObservers(event interface{})
}
