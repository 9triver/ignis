package platform

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils"
)

type Platform struct {
	// Root context of platform
	ctx context.Context
	// Main actor system of the platform
	sys *actor.ActorSystem
	// Control connection manager
	cm remote.ControllerManager
	// Executor connection manager
	em remote.ExecutorManager
	// Deployer
	dp task.Deployer
	// Application infos
	appInfos map[string]*ApplicationInfo
	// Controller actor refs
	controllerActorRefs map[string]*proto.ActorRef
}

func (p *Platform) Run() error {
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	go func() {
		logrus.Infof("Controller Manager listening on %s", p.cm.Addr())
		if err := p.cm.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		logrus.Infof("Executor Manager listening on %s", p.em.Addr())
		if err := p.em.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		logrus.Info("Waiting for new client connection...")
		for {
			ctrlr := p.cm.NextController()
			if ctrlr == nil {
				continue
			}
			logrus.Info("New client is connected")
			msg := <-ctrlr.RecvChan()
			if msg.Type == controller.CommandType_FR_REGISTER_REQUEST {
				req := msg.GetRegisterRequest()
				if req == nil {
					logrus.Error("Register request is nil")
					continue
				}
				appID := req.GetApplicationID()
				if _, ok := p.appInfos[appID]; ok {
					logrus.Errorf("Application ID %s is conflicted", appID)
					continue
				}
				appInfo := NewApplicationInfo(appID)
				actorRef := control.SpawnTaskControllerV2(p.sys.Root, appID, p.dp, appInfo, ctrlr, func() {
					// delete(p.appInfos, appID)
				})
				p.appInfos[appID] = appInfo
				p.controllerActorRefs[appID] = actorRef
				logrus.Infof("Application %s is registered", appID)

				ack := controller.NewAck(nil)
				ctrlr.SendChan() <- ack
			} else {
				logrus.Errorf("The first message %s is not register request", msg.Type)
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

func NewPlatform(ctx context.Context, rpcAddr string, dp task.Deployer) *Platform {
	opt := utils.WithLogger()
	ipcAddr := "ipc://" + path.Join(configs.StoragePath, "em-ipc")

	sys := actor.NewActorSystem(opt)
	em := ipc.NewManager(ipcAddr)

	if dp == nil {
		vm, err := functions.NewVenvManager(ctx, em)
		if err != nil {
			panic(err)
		}

		dp = task.NewVenvMgrDeployer(vm)
	}

	return &Platform{
		ctx:                 ctx,
		sys:                 sys,
		cm:                  rpc.NewManager(rpcAddr),
		em:                  em,
		dp:                  dp,
		appInfos:            make(map[string]*ApplicationInfo),
		controllerActorRefs: make(map[string]*proto.ActorRef),
	}
}

// GetControllerActorRef returns the actor ref of the controller of the application.
func (p *Platform) GetApplicationDAG(appID string) *controller.DAG {
	logrus.Infof("GetApplicationDAG: %s", appID)
	logrus.Infof("appInfos: %v", p.appInfos)
	if _, ok := p.appInfos[appID]; !ok {
		return nil
	}
	return p.appInfos[appID].GetDAG()
}

// GetApplicationInfo returns the ApplicationInfo for the given application ID
func (p *Platform) GetApplicationInfo(appID string) *ApplicationInfo {
	if appInfo, ok := p.appInfos[appID]; ok {
		return appInfo
	}
	return nil
}

// NodeState represents the state of a DAG node
type NodeState struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // "control" or "data"
	Done     bool      `json:"done"`
	Ready    bool      `json:"ready"`
	UpdateAt time.Time `json:"updateAt"`
}

// DAGStateChangeEvent represents a DAG state change event
type DAGStateChangeEvent struct {
	AppID     string     `json:"appId"`
	NodeID    string     `json:"nodeId"`
	NodeState *NodeState `json:"nodeState"`
	Timestamp time.Time  `json:"timestamp"`
}

// StateChangeObserver defines the interface for observing state changes
type StateChangeObserver interface {
	OnDAGStateChanged(event *DAGStateChangeEvent)
}

type ApplicationInfo struct {
	ID         string
	dag        *controller.DAG
	nodeStates map[string]*NodeState
	observers  []StateChangeObserver
	mutex      sync.RWMutex
}

// NewApplicationInfo creates a new ApplicationInfo instance
func NewApplicationInfo(appID string) *ApplicationInfo {
	return &ApplicationInfo{
		ID:         appID,
		nodeStates: make(map[string]*NodeState),
		observers:  make([]StateChangeObserver, 0),
	}
}

// SetDAG sets the DAG of the current application.
func (a *ApplicationInfo) SetDAG(dag *controller.DAG) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.dag = dag

	// Initialize node states from DAG
	if a.nodeStates == nil {
		a.nodeStates = make(map[string]*NodeState)
	}

	for _, node := range dag.Nodes {
		var nodeState *NodeState
		if node.Type == "ControlNode" && node.GetControlNode() != nil {
			cn := node.GetControlNode()
			nodeState = &NodeState{
				ID:       cn.Id,
				Type:     "control",
				Done:     cn.Done,
				Ready:    true, // Control nodes are ready by default
				UpdateAt: time.Now(),
			}
		} else if node.Type == "DataNode" && node.GetDataNode() != nil {
			dn := node.GetDataNode()
			nodeState = &NodeState{
				ID:       dn.Id,
				Type:     "data",
				Done:     dn.Done,
				Ready:    dn.Ready,
				UpdateAt: time.Now(),
			}
		}

		if nodeState != nil {
			a.nodeStates[nodeState.ID] = nodeState
		}
	}
}

// GetDAG returns the DAG of the current application.
func (a *ApplicationInfo) GetDAG() *controller.DAG {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.dag
}

// MarkNodeDone marks a node as done and notifies observers
func (a *ApplicationInfo) MarkNodeDone(nodeID string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if nodeState, exists := a.nodeStates[nodeID]; exists {
		nodeState.Done = true
		nodeState.UpdateAt = time.Now()

		// Update the DAG node state as well
		if a.dag != nil {
			for _, node := range a.dag.Nodes {
				if node.Type == "ControlNode" && node.GetControlNode() != nil && node.GetControlNode().Id == nodeID {
					node.GetControlNode().Done = true
				} else if node.Type == "DataNode" && node.GetDataNode() != nil && node.GetDataNode().Id == nodeID {
					node.GetDataNode().Done = true
				}
			}
		}

		// Notify observers
		event := &DAGStateChangeEvent{
			AppID:     a.ID,
			NodeID:    nodeID,
			NodeState: nodeState,
			Timestamp: time.Now(),
		}

		for _, observer := range a.observers {
			go observer.OnDAGStateChanged(event)
		}
	}
}

// GetNodeStates returns all node states
func (a *ApplicationInfo) GetNodeStates() map[string]*NodeState {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Return a copy to avoid race conditions
	states := make(map[string]*NodeState)
	for k, v := range a.nodeStates {
		stateCopy := *v
		states[k] = &stateCopy
	}
	return states
}

// AddObserver adds a state change observer
func (a *ApplicationInfo) AddObserver(observer StateChangeObserver) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.observers = append(a.observers, observer)
}

// RemoveObserver removes a state change observer
func (a *ApplicationInfo) RemoveObserver(observer StateChangeObserver) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for i, obs := range a.observers {
		if obs == observer {
			a.observers = append(a.observers[:i], a.observers[i+1:]...)
			break
		}
	}
}
