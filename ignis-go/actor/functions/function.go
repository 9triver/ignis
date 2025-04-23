package functions

import (
	"reflect"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

// Function	defines a customized task handler for Actor.
// Implementing this could provide support for various function calls, e.g. Go func, IPC, etc.
type Function interface {
	Name() string
	Params() utils.Set[string]
	Call(params map[string]messages.Object) (messages.Object, error)
	Language() proto.Language
}

type FuncDec struct {
	name   string
	params utils.Set[string]
}

func (f FuncDec) Name() string {
	return f.name
}

func (f FuncDec) Params() utils.Set[string] {
	return f.params
}

func DeclareTyped[T, R any](name string) FuncDec {
	var tmp T
	t := reflect.TypeOf(tmp)
	params := utils.MakeSet[string]()
	for i := range t.NumField() {
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
