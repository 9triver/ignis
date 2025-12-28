package task

import (
	"fmt"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
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
	DeployGoFunc(ctx actor.Context, appId string, f *controller.AppendGo, store *proto.StoreRef) ([]*proto.ActorInfo, error)
	DeployPyFunc(ctx actor.Context, appId string, f *controller.AppendPyFunc, store *proto.StoreRef) ([]*proto.ActorInfo, error)
}

type Config struct {
	RPCAddr  string
	Deployer Deployer
}

// LocalDeployer 是一个部署器，用于部署 Python 函数到 Venv 环境（原始默认实现）
type LocalDeployer struct {
	vm *python.VenvManager
}

func NewVenvMgrDeployer(vm *python.VenvManager) *LocalDeployer {
	return &LocalDeployer{
		vm: vm,
	}
}

func (LocalDeployer) deployAsActors(
	ctx actor.Context,
	f functions.Function,
	store *proto.StoreRef,
	appId, name string, replicas int,
) []*proto.ActorInfo {
	infos := make([]*proto.ActorInfo, replicas)

	for i := range replicas {
		name := fmt.Sprintf("%s:%s-%d", appId, name, i)
		props := compute.NewActor(name, f, store.PID)
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
		infos[i] = info
	}

	return infos
}

func (d *LocalDeployer) DeployPyFunc(ctx actor.Context, appId string, f *controller.AppendPyFunc, store *proto.StoreRef) ([]*proto.ActorInfo, error) {
	pyFunc, err := functions.NewPy(d.vm, f.Name, f.Params, f.Venv, f.Requirements, f.PickledObject, f.Language)
	if err != nil {
		return nil, err
	}

	return d.deployAsActors(ctx, pyFunc, store, appId, f.Name, int(f.Replicas)), nil
}

func (d *LocalDeployer) DeployGoFunc(ctx actor.Context, appId string, f *controller.AppendGo, store *proto.StoreRef) ([]*proto.ActorInfo, error) {
	goFunc, err := functions.ImplGo(f.Name, f.Params, f.Code, f.Language)
	if err != nil {
		return nil, err
	}

	return d.deployAsActors(ctx, goFunc, store, appId, f.Name, int(f.Replicas)), nil
}
