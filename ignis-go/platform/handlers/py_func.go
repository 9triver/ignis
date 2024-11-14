package handlers

import (
	"actors/platform/handlers/ipc"
	"actors/platform/store"
	"actors/platform/utils"
	"actors/proto"
	"fmt"
	"strings"
)

type PyFunction struct {
	FuncDec
	venv *ipc.VirtualEnv
}

func NewPyFunc(
	name string,
	params []string,
	venv string,
	packages []string,
	pickledObj []byte,
) (*PyFunction, error) {
	env, err := ipc.GetVenvManager().GetVenv(venv)
	if err != nil {
		return nil, err
	}

	addHandler := ipc.MakeAddHandler(name, pickledObj, store.LangJson, nil)
	env.Tell(ipc.MakeRouterMessage(addHandler))

	set := utils.MakeSet[string]()
	for _, p := range params {
		set.Add(p)
	}

	return &PyFunction{
		FuncDec: Declare(name, params),
		venv:    env,
	}, nil
}

func ImplPyFunc(
	def FuncDec,
	venv string,
	packages []string,
	pickledObj []byte,
) (*PyFunction, error) {
	return NewPyFunc(def.Name(), def.Params(), venv, packages, pickledObj)
}

func (f *PyFunction) Call(params map[string]*store.Object, s *store.Store) (*store.Object, error) {
	encodedParams := make(map[string]*proto.EncodedObject)
	for k, v := range params {
		switch v.Language {
		case store.LangJson, store.LangPython:
			encoded, err := v.Encode()
			if err != nil {
				return nil, fmt.Errorf("param %s: %w", k, err)
			}
			encodedParams[k] = encoded
		default:
			return nil, fmt.Errorf("param %s: unsupported language %s", k, v.Language)
		}
	}

	segs := strings.Split(f.Name(), ".")
	var obj, method string
	if len(segs) >= 2 {
		obj, method = segs[0], segs[1]
	} else {
		obj, method = f.Name(), ""
	}
	exec := ipc.MakeExecute(obj, method, encodedParams)
	result, err := f.venv.Execute(exec)
	if err != nil {
		return nil, err
	}
	return s.AddObject(result), nil
}

func (f *PyFunction) Language() store.Language {
	return store.LangJson
}

type PyInstance struct {
	venv    *ipc.VirtualEnv
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
	name string,
	methods map[string][]string,
	venv string,
	packages []string,
	pickledObj []byte,
) (*PyInstance, error) {
	env, err := ipc.GetVenvManager().GetVenv(venv)
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

	addHandler := ipc.MakeAddHandler(name, pickledObj, store.LangJson, names)
	env.Tell(ipc.MakeRouterMessage(addHandler))

	return &PyInstance{venv: env, name: name, methods: funcs}, nil
}
