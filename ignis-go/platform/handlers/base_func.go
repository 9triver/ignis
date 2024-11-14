package handlers

import (
	"actors/platform/store"
	"actors/platform/utils"
	"reflect"
)

type Function interface {
	Name() string
	Params() []string
	ParamSet() utils.Set[string]
	Call(params map[string]*store.Object, store *store.Store) (*store.Object, error)
	Language() store.Language
}

type DeployLocal interface {
	Function
}

type FuncDec struct {
	name   string
	params utils.Set[string]
}

func (f FuncDec) Name() string {
	return f.name
}

func (f FuncDec) Params() []string {
	return f.params.Values()
}

func (f FuncDec) ParamSet() utils.Set[string] {
	return f.params
}

func DeclareTyped[T, R any](name string) FuncDec {
	var tmp T
	t := reflect.TypeOf(tmp)
	params := utils.MakeSet[string]()
	for i := 0; i < t.NumField(); i++ {
		params.Add(t.Field(i).Name)
	}
	return FuncDec{name, params}
}

func Declare(name string, params []string) FuncDec {
	set := utils.MakeSet[string]()
	for _, p := range params {
		set.Add(p)
	}
	return FuncDec{name, set}
}
