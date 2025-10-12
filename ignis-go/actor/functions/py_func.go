package functions

import (
	"fmt"
	"strings"
	"time"

	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils/errors"
)

type PyFunction struct {
	FuncDec
	venv     *VirtualEnv
	language proto.Language
}

func NewPy(
	manager *VenvManager,
	name string,
	params []string,
	venv string,
	packages []string,
	pickledObj []byte,
	language proto.Language,
) (*PyFunction, error) {
	dec := Declare(name, params)
	return ImplPy(manager, dec, venv, packages, pickledObj, language)
}

func ImplPy(
	manager *VenvManager,
	dec FuncDec,
	venv string,
	packages []string,
	pickledObj []byte,
	language proto.Language,
) (*PyFunction, error) {
	name := dec.Name()
	env, err := manager.GetVenv(venv, packages...)
	if err != nil {
		return nil, errors.WrapWith(err, "%s: error creating impl", name)
	}

	addHandler := executor.NewAddHandler(venv, name, pickledObj, language, nil)
	env.Send(addHandler)

	return &PyFunction{
		FuncDec:  dec,
		venv:     env,
		language: language,
	}, nil
}

func NewPyClass(
	manager *VenvManager,
	objName string,
	methods []string,
	params [][]string,
	venv string,
	packages []string,
	pickledObj []byte,
	language proto.Language,
) ([]*PyFunction, error) {
	env, err := manager.GetVenv(venv, packages...)
	if err != nil {
		return nil, errors.WrapWith(err, "%s: error creating impl", objName)
	}

	addHandler := executor.NewAddHandler(venv, objName, pickledObj, language, methods)
	env.Send(addHandler)

	var functions []*PyFunction
	for i, method := range methods {
		functions = append(functions, &PyFunction{
			FuncDec: FuncDec{
				name:   fmt.Sprintf("%s.%s", objName, method),
				params: params[i],
			},
			venv:     env,
			language: language,
		})
	}

	return functions, nil
}

func (f *PyFunction) Call(params map[string]objects.Interface) (objects.Interface, error) {
	segs := strings.Split(f.Name(), ".")
	var obj, method string
	if len(segs) >= 2 {
		obj, method = segs[0], segs[1]
	} else {
		obj, method = f.Name(), ""
	}
	result, err := f.venv.Execute(obj, method, params).Result()
	if err != nil {
		return nil, errors.WrapWith(err, "%s: execution failed", f.name)
	}
	return result, nil
}

func (f *PyFunction) TimedCall(params map[string]objects.Interface) (time.Duration, objects.Interface, error) {
	start := time.Now()
	obj, err := f.Call(params)
	return time.Since(start), obj, err
}

func (f *PyFunction) Language() proto.Language {
	return f.language
}
