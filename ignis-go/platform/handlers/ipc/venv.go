package ipc

import (
	"actors/platform/store"
	"actors/platform/system"
	"actors/platform/utils"
	proto_ipc "actors/proto/ipc"

	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
)

const (
	kDefaultVenvStorage = "actor-platform"
	venvMetadata        = "envs.json"
	venvIPCAddr         = "python-ipc"
	venvStart           = "executor.py"

	VenvSystem = ""
)

//go:embed venv_template.py
var execTemplate string

var (
	VenvStorage = func() string {
		home, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}
		return path.Join(home, kDefaultVenvStorage)
	}()
)

type VenvCall struct {
	Venv string
	Call *proto_ipc.Execute
}

type VirtualEnv struct {
	mu sync.Mutex

	Name     string   `json:"name"`
	Exec     string   `json:"exec"`
	Packages []string `json:"packages"`
}

func (v *VirtualEnv) Interpreter() string {
	return v.Exec
}

func (v *VirtualEnv) pip() string {
	return path.Join(VenvStorage, v.Name, "bin", "pip")
}

func (v *VirtualEnv) AddPackages(p ...string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	pkgSet := utils.MakeSetFromSlice(v.Packages)
	for _, pkg := range p {
		if pkgSet.Contains(pkg) {
			continue
		}
		err := exec.Command(v.pip(), "install", pkg).Run()
		if err != nil {
			return fmt.Errorf("venv %s: failed to install package %s: %w", v.Name, p, err)
		}
		v.Packages = append(v.Packages, pkg)
		pkgSet.Add(pkg)
	}
	return nil
}

func (v *VirtualEnv) Execute(cmd *proto_ipc.Execute) (*store.Object, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return GetVenvManager().Handler().Request(v.Name, cmd)
}

func (v *VirtualEnv) Tell(cmd *proto_ipc.RouterMessage) {
	GetVenvManager().Handler().Tell(v.Name, cmd)
}

func (v *VirtualEnv) Serve(ipcAddr string) <-chan error {
	cmd := exec.Command(v.Exec, path.Join(VenvStorage, venvStart), "--ipc", ipcAddr, "--venv", v.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ch := GetVenvManager().Handler().Register(v.Name)
	go func() {
		if err := cmd.Run(); err != nil {
			ch <- err
		}
		close(ch)
	}()
	return ch
}

type VenvManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler *Handler

	SystemExec string        `json:"system_py"`
	Envs       []*VirtualEnv `json:"envs"`
}

func (m *VenvManager) Handler() *Handler {
	return m.handler
}

func (m *VenvManager) IPCAddr() string {
	return "ipc://" + path.Join(VenvStorage, venvIPCAddr)
}

func (m *VenvManager) AddVenv(venv *VirtualEnv) {
	m.Envs = append(m.Envs, venv)
}

func (m *VenvManager) CreateVenv(name string) (*VirtualEnv, error) {
	venvPath := path.Join(VenvStorage, name)

	if err := os.MkdirAll(venvPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create venv %s: %w", name, err)
	}

	err := exec.Command("python3", "-m", "venv", venvPath).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to create venv %s: %w", name, err)
	}

	return &VirtualEnv{
		Name:     name,
		Exec:     path.Join(venvPath, "bin", "python"),
		Packages: []string{"pyzmq"},
	}, nil
}

func (m *VenvManager) GetVenv(name string, requirements ...string) (*VirtualEnv, error) {
	if name == VenvSystem {
		if len(requirements) > 0 {
			return nil, fmt.Errorf("system venv does not support installing packages")
		}
		return &VirtualEnv{Name: VenvSystem, Exec: manager.SystemExec}, nil
	}

	for _, env := range m.Envs {
		if env.Name == name {
			return env, nil
		}
	}

	env, err := m.CreateVenv(name)
	if err != nil {
		return nil, err
	}

	err = env.AddPackages(requirements...)
	if err != nil {
		return nil, err
	}
	m.Envs = append(m.Envs, env)
	return env, <-env.Serve(m.IPCAddr())
}

func (m *VenvManager) save() error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(VenvStorage, venvMetadata), data, os.ModePerm)
}

func (m *VenvManager) Close() error {
	m.cancel()
	return m.save()
}

func (m *VenvManager) Run() {
	logger := system.Logger()
	logger.Debug("starting venv manager", "ipc", m.IPCAddr())
	m.handler = NewHandler(m.ctx, m.IPCAddr(), system.ExecutionTimeout)

	go func() {
		errors := m.handler.Run()
		for {
			select {
			case <-m.ctx.Done():
				return
			case err := <-errors:
				logger.Error("venv manager error", "msg", err.Error())
			}
		}
	}()
}

func createManager() *VenvManager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &VenvManager{
		ctx:    ctx,
		cancel: cancel,
	}

	if _, err := os.Stat(VenvStorage); os.IsNotExist(err) {
		if err := os.MkdirAll(VenvStorage, os.ModePerm); err != nil {
			panic(err)
		}

		if err := os.WriteFile(path.Join(VenvStorage, venvStart), []byte(execTemplate), os.ModePerm); err != nil {
			panic(err)
		}

		return m
	}

	metadata := path.Join(VenvStorage, "envs.json")
	data, err := os.ReadFile(metadata)
	if err != nil {
		return m
	}

	tmp := &VenvManager{ctx: ctx, cancel: cancel}
	if err := json.Unmarshal(data, tmp); err != nil {
		return tmp
	}
	return m
}

var (
	manager *VenvManager
	once    sync.Once
)

func GetVenvManager() *VenvManager {
	once.Do(func() {
		manager = createManager()
		manager.SystemExec = "/usr/bin/python3"
	})
	return manager
}
