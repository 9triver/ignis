// Package functions 提供了函数调用的封装和适配功能
// 该文件实现了 Python 虚拟环境管理器，用于管理多个独立的 Python 虚拟环境
package functions

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/transport/ipc"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// Python 虚拟环境相关常量
const (
	venvStorageName = "actor-platform"      // 虚拟环境存储目录名
	venvMetadata    = ".envs.json"          // 虚拟环境元数据文件名
	venvStart       = "__actor_executor.py" // Python 执行器启动脚本名
	pythonExec      = "python3"             // 系统 Python 可执行文件名
)

// venvPath 是虚拟环境的存储根路径
// 位于用户配置目录下的 actor-platform 子目录中
var (
	venvPath     string
	venvPathOnce sync.Once
	venvPathErr  error
)

// getVenvPath 延迟初始化虚拟环境存储路径
// 使用 sync.Once 确保只初始化一次，避免包级 panic
func getVenvPath() (string, error) {
	venvPathOnce.Do(func() {
		home, err := os.UserConfigDir()
		if err != nil {
			venvPathErr = errors.WrapWith(err, "failed to get user config dir")
			return
		}
		dir := path.Join(home, venvStorageName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			venvPathErr = errors.WrapWith(err, "failed to create venv storage dir")
			return
		}
		venvPath = dir
	})
	return venvPath, venvPathErr
}

// VenvManager 是 Python 虚拟环境管理器
// 负责创建、管理和持久化多个独立的 Python 虚拟环境
// 每个虚拟环境可以有自己独立的包依赖和配置
//
// 主要功能:
//   - 创建和管理多个独立的 Python 虚拟环境
//   - 自动安装虚拟环境所需的包依赖
//   - 持久化虚拟环境配置（保存到 JSON 文件）
//   - 管理虚拟环境的远程执行器
//   - 线程安全的虚拟环境访问
type VenvManager struct {
	mu      sync.Mutex             // 互斥锁，保护并发访问
	ctx     context.Context        // 上下文，用于生命周期管理
	cancel  context.CancelFunc     // 取消函数，用于关闭管理器
	em      transport.ExecutorManager // 远程执行器管理器
	started map[string]bool        // 记录已启动的虚拟环境

	SystemExec string                 `json:"system_py"` // 系统 Python 可执行文件路径
	Envs       map[string]*VirtualEnv `json:"envs"`      // 虚拟环境映射表，key 为环境名
}

// ExecutorManager 返回远程执行器管理器
// 返回值:
//   - transport.ExecutorManager: 执行器管理器实例
func (m *VenvManager) ExecutorManager() transport.ExecutorManager {
	return m.em
}

// addVenv 将虚拟环境添加到管理器中
// 参数:
//   - venv: 要添加的虚拟环境实例
func (m *VenvManager) addVenv(venv *VirtualEnv) {
	m.Envs[venv.Name] = venv
}

// template 根据执行器管理器类型返回对应的 Python 执行器模板代码
// 返回值:
//   - string: 模板代码内容
//   - bool: 是否找到对应的模板（true 表示找到）
func (m *VenvManager) template() (string, bool) {
	switch m.em.Type() {
	case transport.IPC:
		return ipc.PythonExecutorTemplate, true
	default:
		return "", false
	}
}

// createEnv 创建一个新的 Python 虚拟环境
// 参数:
//   - name: 虚拟环境名称
//
// 返回值:
//   - *VirtualEnv: 创建的虚拟环境实例
//   - error: 创建过程中的错误
//
// 执行流程:
//  1. 在 venvPath 下创建以 name 命名的目录
//  2. 使用 python3 -m venv 命令创建虚拟环境（带超时控制）
//  3. 将 Python 执行器模板脚本写入虚拟环境目录
//  4. 创建并返回 VirtualEnv 实例，默认安装 pyzmq 包
func (m *VenvManager) createEnv(name string) (*VirtualEnv, error) {
	basePath, err := getVenvPath()
	if err != nil {
		return nil, err
	}

	envPath := path.Join(basePath, name)

	if err := os.MkdirAll(envPath, 0755); err != nil {
		return nil, errors.WrapWith(err, "venv %s: path creation failed", name)
	}

	// 使用超时控制创建虚拟环境
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-m", "venv", envPath)
	if err := cmd.Run(); err != nil {
		return nil, errors.WrapWith(err, "venv %s: venv creation failed", name)
	}

	if template, ok := m.template(); !ok {
		return nil, errors.Format("venv %s: no template is found for python", name)
	} else if err := os.WriteFile(path.Join(envPath, venvStart), []byte(template), 0644); err != nil {
		return nil, errors.WrapWith(err, "venv %s: template creation failed", name)
	}

	return &VirtualEnv{
		ctx:      m.ctx,
		handler:  m.em.NewExecutor(m.ctx, name),
		futures:  make(map[string]utils.Future[object.Interface]),
		streams:  make(map[string]*object.Stream),
		Name:     name,
		Exec:     path.Join(envPath, "bin", "python"),
		Packages: []string{"pyzmq"},
	}, nil
}

// GetVenv 获取或创建指定名称的虚拟环境
// 参数:
//   - name: 虚拟环境名称
//   - requirements: 可选的包依赖列表
//
// 返回值:
//   - *VirtualEnv: 虚拟环境实例
//   - error: 操作过程中的错误
//
// 行为说明:
//   - 如果虚拟环境已存在，检查并安装新的依赖，然后启动它
//   - 如果虚拟环境不存在，创建新的虚拟环境并安装指定的依赖包
//   - 该方法是线程安全的（使用互斥锁保护）
func (m *VenvManager) GetVenv(name string, requirements ...string) (*VirtualEnv, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if env, ok := m.Envs[name]; ok {
		// 检查是否有新的依赖需要安装
		if len(requirements) > 0 {
			if err := env.AddPackages(requirements...); err != nil {
				return nil, err
			}
		}
		// 只在未启动时才启动
		if !m.started[name] {
			env.Run(m.em.Addr())
			m.started[name] = true
		}
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
	m.started[name] = true
	return env, nil
}

// save 将管理器的状态持久化到 JSON 文件
// 返回值:
//   - error: 保存过程中的错误
//
// 保存内容包括:
//   - SystemExec: 系统 Python 路径
//   - Envs: 所有虚拟环境的配置（名称、路径、包依赖等）
//
// 使用原子写入策略：先写入临时文件，然后重命名，避免写入失败导致文件损坏
func (m *VenvManager) save() error {
	basePath, err := getVenvPath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(m)
	if err != nil {
		return errors.WrapWith(err, "failed to marshal manager state")
	}

	// 先写入临时文件，然后原子重命名
	tmpPath := path.Join(basePath, venvMetadata+".tmp")
	finalPath := path.Join(basePath, venvMetadata)

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return errors.WrapWith(err, "failed to write temporary metadata file")
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return errors.WrapWith(err, "failed to rename metadata file")
	}

	return nil
}

// Close 关闭管理器并保存状态
// 返回值:
//   - error: 关闭或保存过程中的错误
//
// 执行操作:
//  1. 调用 cancel 函数取消所有虚拟环境的上下文
//  2. 将管理器状态保存到磁盘
func (m *VenvManager) Close() error {
	m.cancel()
	return m.save()
}

// NewVenvManager 创建一个新的虚拟环境管理器实例
// 参数:
//   - ctx: 上下文，用于生命周期管理
//   - manager: 远程执行器管理器
//
// 返回值:
//   - *VenvManager: 管理器实例
//   - error: 创建过程中的错误
//
// 初始化流程:
//  1. 创建带取消功能的子上下文
//  2. 初始化管理器的基本字段
//  3. 尝试从磁盘加载之前保存的虚拟环境配置
//  4. 如果加载成功，重新初始化所有虚拟环境的运行时字段（上下文、执行器等）
func NewVenvManager(ctx context.Context, manager transport.ExecutorManager) (*VenvManager, error) {
	// 确保虚拟环境路径已初始化
	basePath, err := getVenvPath()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	m := &VenvManager{
		ctx:        ctx,
		cancel:     cancel,
		em:         manager,
		started:    make(map[string]bool),
		SystemExec: pythonExec,
		Envs:       make(map[string]*VirtualEnv),
	}

	// 尝试从磁盘加载之前保存的虚拟环境配置
	if data, err := os.ReadFile(path.Join(basePath, venvMetadata)); err == nil {
		if err := json.Unmarshal(data, m); err != nil {
			return nil, errors.WrapWith(err, "venv manager: error reading metadata")
		}

		// 重新初始化所有虚拟环境的运行时字段
		for _, env := range m.Envs {
			env.started = false
			env.ctx = ctx
			env.handler = manager.NewExecutor(ctx, env.Name)
			env.futures = make(map[string]utils.Future[object.Interface])
			env.streams = make(map[string]*object.Stream)
		}
	}

	return m, nil
}
