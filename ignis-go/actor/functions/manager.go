package functions

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

const (
	venvStorageName = "actor-platform"
	venvMetadata    = ".envs.json"
	venvStart       = "__actor_executor.py"
	pythonExec      = "/home/spark4862/anaconda3/envs/demo/bin/python"
)

var venvPath = func() string {
	home, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	dir := path.Join(home, venvStorageName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	return dir
}()

type VenvManager struct {
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	em      remote.ExecutorManager
	started map[string]bool

	SystemExec string                 `json:"system_py"`
	Envs       map[string]*VirtualEnv `json:"envs"`
}

func (m *VenvManager) ExecutorManager() remote.ExecutorManager {
	return m.em
}

func (m *VenvManager) addVenv(venv *VirtualEnv) {
	m.Envs[venv.Name] = venv
}

func (m *VenvManager) template() (string, bool) {
	switch m.em.Type() {
	case remote.IPC:
		return ipc.PythonExecutorTemplate, true
	default:
		return "", false
	}
}

func (m *VenvManager) createEnv(name string) (*VirtualEnv, error) {
	venvPath := path.Join(venvPath, name)

	if err := os.MkdirAll(venvPath, 0o755); err != nil {
		return nil, errors.WrapWith(err, "venv %s: path creation failed", name)
	}

	if err := exec.Command(pythonExec, "-m", "venv", venvPath).Run(); err != nil {
		return nil, errors.WrapWith(err, "venv %s: venv creation failed", name)
	}

	if template, ok := m.template(); !ok {
		return nil, errors.Format("venv %s: no template is found for python", name)
	} else if err := os.WriteFile(path.Join(venvPath, venvStart), []byte(template), 0o644); err != nil {
		return nil, errors.WrapWith(err, "venv %s: template creation failed", name)
	}

	return &VirtualEnv{
		ctx:      m.ctx,
		handler:  m.em.NewExecutor(m.ctx, name),
		futures:  make(map[string]utils.Future[objects.Interface]),
		streams:  make(map[string]*objects.Stream),
		Name:     name,
		Exec:     path.Join(venvPath, "bin", "python"),
		Packages: []string{"pyzmq"},
	}, nil
}

func (m *VenvManager) GetVenv(name string, requirements ...string) (*VirtualEnv, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if env, ok := m.Envs[name]; ok {
		env.Run(m.em.Addr())
		return env, nil
	}

	env, err := m.createEnv(name)
	if err != nil {
		return nil, err
	}

	if err = env.AddPackages(requirements...); err != nil {
		return nil, err
	}

	m.addVenv(env)

	env.Run(m.em.Addr())
	return env, nil
}

func (m *VenvManager) save() error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(venvPath, venvMetadata), data, os.ModePerm)
}

func (m *VenvManager) Close() error {
	m.cancel()
	return m.save()
}

func NewVenvManager(ctx context.Context, manager remote.ExecutorManager) (*VenvManager, error) {
	ctx, cancel := context.WithCancel(ctx)
	m := &VenvManager{
		ctx:        ctx,
		cancel:     cancel,
		em:         manager,
		started:    make(map[string]bool),
		SystemExec: pythonExec,
		Envs:       make(map[string]*VirtualEnv),
	}

	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		if err := os.MkdirAll(venvPath, os.ModePerm); err != nil {
			return nil, errors.WrapWith(err, "venv manager: error creating dir")
		}
		return m, nil
	}

	if data, err := os.ReadFile(path.Join(venvPath, venvMetadata)); err != nil {
		return m, nil
	} else if err := json.Unmarshal(data, m); err != nil {
		return nil, errors.WrapWith(err, "venv manager: error reading metadata")
	}

	for _, env := range m.Envs {
		env.started = false
		env.ctx = ctx
		env.handler = manager.NewExecutor(ctx, env.Name)
		env.futures = make(map[string]utils.Future[objects.Interface])
		env.streams = make(map[string]*objects.Stream)
	}
	return m, nil
}
