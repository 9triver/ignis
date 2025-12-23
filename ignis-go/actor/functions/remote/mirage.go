package remote

import (
	_ "embed"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/9triver/ignis/utils/errors"
)

var (
	//go:embed "mirage/build_template.sh"
	buildTemplate []byte

	//go:embed "mirage/config.ml"
	configFile []byte

	//go:embed "mirage/unikernel.ml"
	unikernelFile []byte
)

type Mirage struct {
	name     string
	manager  *Manager
	handlers string
	target   string
}

var (
	miragePath     string
	miragePathOnce sync.Once
	miragePathErr  error
)

func getUnikernelPath() (string, error) {
	miragePathOnce.Do(func() {
		home, err := os.UserConfigDir()
		if err != nil {
			miragePathErr = errors.WrapWith(err, "failed to get user config dir")
			return
		}
		dir := path.Join(home, "unikernels")
		if err := os.MkdirAll(dir, 0755); err != nil {
			miragePathErr = errors.WrapWith(err, "failed to create venv storage dir")
			return
		}
		miragePath = dir
	})
	return miragePath, miragePathErr
}

func (m *Mirage) Build() error {
	basePath, err := getUnikernelPath()
	if err != nil {
		return err
	}

	workdir := path.Join(basePath, m.name)
	if err := os.MkdirAll(workdir, 0755); err != nil {
		return errors.WrapWith(err, "venv %s: path creation failed", m.name)
	}

	executable := path.Join(workdir, "dist", "ws-handler")
	if _, err := os.Stat(executable); err == nil || os.IsExist(err) {
		return nil
	}

	// write config.ml, unikernel.ml
	if err := os.WriteFile(path.Join(workdir, "config.ml"), configFile, 0644); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(workdir, "unikernel.ml"), unikernelFile, 0644); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(workdir, "handlers.ml"), []byte(m.handlers), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(workdir, "build.sh"), buildTemplate, 0755); err != nil {
		return err
	}

	cmd := exec.Command("./build.sh", m.target)
	cmd.Dir = workdir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func (m *Mirage) Run(connId string) error {
	basePath, err := getUnikernelPath()
	if err != nil {
		return err
	}

	workdir := path.Join(basePath, m.name)
	executable := path.Join(workdir, "dist", "ws-handler")

	if _, err := os.Stat(executable); os.IsNotExist(err) {
		return errors.WrapWith(err, "unikernel: not available")
	}

	switch m.target {
	case "unix":
		cmd := exec.Command(
			path.Join("dist", "ws-handler"),
			"--id", connId,
			"--uri", m.manager.Endpoint(),
		)
		cmd.Dir = workdir
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	// case "hvt":
	default:
		return errors.New("unsupported target")
	}
}

func NewMirage(
	name string,
	manager *Manager,
	handlers string,
	target string,
) *Mirage {
	return &Mirage{
		name:     name,
		manager:  manager,
		handlers: handlers,
		target:   target,
	}
}
