package functions

import (
	"github.com/asynkron/protoactor-go/actor"
	"strings"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils/errors"
)

type PyFunction struct {
	FuncDec
	venv     *python.VirtualEnv
	language proto.Language
}

var _ Function = (*PyFunction)(nil)

func NewPy(
	manager *python.VenvManager,
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
	manager *python.VenvManager,
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

func (f *PyFunction) Call(ctx actor.Context, sessionId string, params map[string]proto.Object) (proto.Object, error) {
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

func (f *PyFunction) Language() proto.Language {
	return f.language
}

type PyInstance struct {
	venv    *python.VirtualEnv
	name    string
	methods map[string]*PyFunction
}

func (actor *PyInstance) Get(method string) (f *PyFunction, ok bool) {
	f, ok = actor.methods[method]
	return
}

func (actor *PyInstance) Functions() (funcs []*PyFunction) {
	for _, f := range actor.methods {
		funcs = append(funcs, f)
	}
	return
}

func NewPyInstance(
	manager *python.VenvManager,
	name string,
	methods map[string][]string,
	venv string,
	packages []string,
	pickledObj []byte,
) (*PyInstance, error) {
	env, err := manager.GetVenv(venv, packages...)
	if err != nil {
		return nil, err
	}

	funcs := make(map[string]*PyFunction)
	var names []string
	for method, params := range methods {
		f := &PyFunction{
			FuncDec: Declare(name+"."+method, params),
			venv:    env,
		}
		names = append(names, method)
		funcs[method] = f
	}

	// TODO: add multiple instance methods

	return &PyInstance{venv: env, name: name, methods: funcs}, nil
}
