package task

import (
	"fmt"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/asynkron/protoactor-go/actor"
)

type Env string

const (
	EnvPython Env = "python"
)

type Resources struct {
	CPU    int64 // CPU milli Cores
	Memory int64 // memory in Bytes
	GPU    int64 // GPU cores
}

type Deployer interface {
	DeployPyFunc(ctx actor.Context, appId string, f *controller.AppendPyFunc, store *proto.StoreRef) ([]*proto.ActorInfo, error)
}

type Config struct {
	RPCAddr  string
	Deployer Deployer
}

// VenvMgrDeployer 是一个部署器，用于部署 Python 函数到 Venv 环境（原始默认实现）
type VenvMgrDeployer struct {
	vm *python.VenvManager
}

func NewVenvMgrDeployer(vm *python.VenvManager) *VenvMgrDeployer {
	return &VenvMgrDeployer{
		vm: vm,
	}
}

func (d *VenvMgrDeployer) DeployPyFunc(ctx actor.Context, appId string, f *controller.AppendPyFunc, store *proto.StoreRef) ([]*proto.ActorInfo, error) {

	pyFunc, err := functions.NewPy(d.vm, f.Name, f.Params, f.Venv, f.Requirements, f.PickledObject, f.Language)
	if err != nil {
		return nil, err
	}

	infos := make([]*proto.ActorInfo, f.Replicas)

	for i := range f.Replicas {
		name := fmt.Sprintf("%s:%s-%d", appId, f.Name, i)
		props := compute.NewActor(name, pyFunc, store.PID)
		pid := ctx.Spawn(props)
		info := &proto.ActorInfo{
			Ref: &proto.ActorRef{
				ID:    name,
				PID:   pid,
				Store: store,
			},
			CalcLatency: 0,
			LinkLatency: 0,
		}
		router.Register(name, pid)
		infos[i] = info
		// group.Push(info)
	}
	return infos, nil
}
