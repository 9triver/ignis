package gofunc

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"plugin"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/utils"
)

//go:embed "common.go.template"
var codeCommon []byte

var errSymbolType = errors.New("symbol type error")

func Compile(name, code string) (WrappedFunc, error) {
	rev := utils.HexMd5Sum([]byte(code))
	cwd := path.Join(configs.StoragePath, "plugins", fmt.Sprintf("plugin.%s-%s", name, rev))

	if err := os.MkdirAll(cwd, 0755); err != nil {
		return nil, err
	}

	file := name + ".so"

	callShell := func(args ...string) error {
		cmd := exec.Command("go", args...)
		cmd.Dir = cwd
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	if err := os.WriteFile(path.Join(cwd, "common.go"), codeCommon, 0644); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path.Join(cwd, "main.go"), []byte(code), 0644); err != nil {
		return nil, err
	}

	if _, err := os.Stat(path.Join(cwd, "go.mod")); os.IsNotExist(err) {
		if err := callShell("mod", "init", name); err != nil {
			return nil, err
		}
	}

	if err := callShell("mod", "tidy"); err != nil {
		return nil, err
	}

	if err := callShell("build", "-buildmode=plugin", "-o", file); err != nil {
		return nil, err
	}

	plug, err := plugin.Open(path.Join(cwd, file))
	if err != nil {
		return nil, err
	}

	sym, err := plug.Lookup("Call")
	if err != nil {
		return nil, err
	}

	impl, ok := sym.(WrappedFunc)
	if !ok {
		return nil, errSymbolType
	}

	return impl, nil
}
