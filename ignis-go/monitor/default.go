package monitor

import (
	"context"

	"github.com/9triver/ignis/proto/controller"
)

// defaultMonitor 默认的空操作监控实现
// 所有方法都是空操作，不执行任何实际逻辑
type defaultMonitor struct{}

// DefaultMonitor 返回默认的空操作监控实例
func DefaultMonitor() Monitor {
	return &defaultMonitor{}
}

// ApplicationMonitor 实现
func (d *defaultMonitor) RegisterApplication(ctx context.Context, appID string, metadata *ApplicationMetadata) error {
	return nil
}
func (d *defaultMonitor) UnregisterApplication(ctx context.Context, appID string) error {
	return nil
}
func (d *defaultMonitor) GetApplicationInfo(ctx context.Context, appID string) (*ApplicationInfo, error) {
	return nil, nil
}
func (d *defaultMonitor) ListApplications(ctx context.Context, filter *ApplicationFilter) ([]*ApplicationInfo, error) {
	return nil, nil
}
func (d *defaultMonitor) UpdateApplicationStatus(ctx context.Context, appID string, status ApplicationStatus) error {
	return nil
}
func (d *defaultMonitor) GetApplicationDAG(ctx context.Context, appID string) (*controller.DAG, error) {
	return nil, nil
}
func (d *defaultMonitor) SetApplicationDAG(ctx context.Context, appID string, dag *controller.DAG) error {
	return nil
}
func (d *defaultMonitor) GetApplicationMetrics(ctx context.Context, appID string) (*ApplicationMetrics, error) {
	return nil, nil
}

// NodeMonitor 实现
func (d *defaultMonitor) GetNodeState(ctx context.Context, appID, nodeID string) (*NodeState, error) {
	return nil, nil
}
func (d *defaultMonitor) GetAllNodeStates(ctx context.Context, appID string) (map[string]*NodeState, error) {
	return nil, nil
}
func (d *defaultMonitor) UpdateNodeState(ctx context.Context, appID, nodeID string, state *NodeState) error {
	return nil
}
func (d *defaultMonitor) MarkNodeReady(ctx context.Context, appID, nodeID string) error {
	return nil
}
func (d *defaultMonitor) MarkNodeRunning(ctx context.Context, appID, nodeID string) error {
	return nil
}
func (d *defaultMonitor) MarkNodeDone(ctx context.Context, appID, nodeID string, result *NodeResult) error {
	return nil
}
func (d *defaultMonitor) MarkNodeFailed(ctx context.Context, appID, nodeID string, err error) error {
	return nil
}
func (d *defaultMonitor) GetNodeMetrics(ctx context.Context, appID, nodeID string) (*NodeMetrics, error) {
	return nil, nil
}
func (d *defaultMonitor) GetNodeDependencies(ctx context.Context, appID, nodeID string) (*NodeDependencies, error) {
	return nil, nil
}
func (d *defaultMonitor) WatchNodeState(ctx context.Context, appID, nodeID string) (<-chan *NodeStateChangeEvent, error) {
	ch := make(chan *NodeStateChangeEvent)
	close(ch)
	return ch, nil
}

// TaskMonitor 实现
func (d *defaultMonitor) RecordTaskStart(ctx context.Context, taskID string, task *TaskInfo) error {
	return nil
}
func (d *defaultMonitor) RecordTaskEnd(ctx context.Context, taskID string, result *TaskResult) error {
	return nil
}
func (d *defaultMonitor) GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error) {
	return nil, nil
}
func (d *defaultMonitor) ListTasks(ctx context.Context, appID string, filter *TaskFilter) ([]*TaskInfo, error) {
	return nil, nil
}
func (d *defaultMonitor) GetTaskMetrics(ctx context.Context, taskID string) (*TaskMetrics, error) {
	return nil, nil
}
func (d *defaultMonitor) CancelTask(ctx context.Context, taskID string) error {
	return nil
}

// ResourceMonitor 实现
func (d *defaultMonitor) RecordResourceUsage(ctx context.Context, appID string, usage *ResourceUsage) error {
	return nil
}
func (d *defaultMonitor) GetResourceUsage(ctx context.Context, appID string) (*ResourceUsage, error) {
	return nil, nil
}
func (d *defaultMonitor) GetResourceHistory(ctx context.Context, appID string, timeRange *TimeRange) ([]*ResourceSnapshot, error) {
	return nil, nil
}
func (d *defaultMonitor) GetWorkerResources(ctx context.Context, workerID string) (*WorkerResources, error) {
	return nil, nil
}
func (d *defaultMonitor) ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*WorkerInfo, error) {
	return nil, nil
}
func (d *defaultMonitor) GetClusterResources(ctx context.Context) (*ClusterResources, error) {
	return nil, nil
}

// MetricsMonitor 实现
func (d *defaultMonitor) RecordMetric(ctx context.Context, metric *Metric) error {
	return nil
}
func (d *defaultMonitor) RecordMetrics(ctx context.Context, metrics []*Metric) error {
	return nil
}
func (d *defaultMonitor) QueryMetrics(ctx context.Context, query *MetricQuery) ([]*Metric, error) {
	return nil, nil
}
func (d *defaultMonitor) GetMetricAggregation(ctx context.Context, query *MetricQuery, aggregation *AggregationConfig) (*AggregationResult, error) {
	return nil, nil
}
func (d *defaultMonitor) ListMetricNames(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (d *defaultMonitor) DeleteMetrics(ctx context.Context, query *MetricQuery) error {
	return nil
}

// EventMonitor 实现
func (d *defaultMonitor) RecordEvent(ctx context.Context, event *Event) error {
	return nil
}
func (d *defaultMonitor) QueryEvents(ctx context.Context, query *EventQuery) ([]*Event, error) {
	return nil, nil
}
func (d *defaultMonitor) Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error) {
	ch := make(chan *Event)
	close(ch)
	return ch, nil
}
func (d *defaultMonitor) Unsubscribe(ctx context.Context, subscriptionID string) error {
	return nil
}
func (d *defaultMonitor) GetEventStatistics(ctx context.Context, query *EventQuery) (*EventStatistics, error) {
	return nil, nil
}

// LifecycleManager 实现
func (d *defaultMonitor) Start(ctx context.Context) error {
	return nil
}
func (d *defaultMonitor) Stop(ctx context.Context) error {
	return nil
}
func (d *defaultMonitor) Status(ctx context.Context) (*MonitorStatus, error) {
	return &MonitorStatus{Running: true}, nil
}
func (d *defaultMonitor) HealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	return &HealthCheckResult{Healthy: true, Checks: make(map[string]bool)}, nil
}
func (d *defaultMonitor) Reset(ctx context.Context) error {
	return nil
}
func (d *defaultMonitor) Export(ctx context.Context, config *ExportConfig) ([]byte, error) {
	return nil, nil
}
func (d *defaultMonitor) Import(ctx context.Context, data []byte) error {
	return nil
}

// ObservableMonitor 实现
func (d *defaultMonitor) AddObserver(observer Observer)     {}
func (d *defaultMonitor) RemoveObserver(observer Observer)  {}
func (d *defaultMonitor) NotifyObservers(event interface{}) {}
