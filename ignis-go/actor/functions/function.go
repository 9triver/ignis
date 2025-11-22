package functions

import (
	"reflect"
	"time"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
)

// Function	defines a customized task handler for Actor.
// Implementing this could provide support for various function calls, e.g. Go func, IPC, etc.
type Function interface {
	Name() string
	Params() []string
	Call(params map[string]object.Interface) (object.Interface, error)
	TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error)
	Language() proto.Language
}

var (
	_ Function = (*GoFunction[any, any])(nil)
	_ Function = (*PyFunction)(nil)
	_ Function = (*RemoteFunction)(nil)
)

type FuncDec struct {
	name   string
	params []string
}

func (f FuncDec) Name() string {
	return f.name
}

func (f FuncDec) Params() []string {
	return f.params
}

func DeclareTyped[T, R any](name string) FuncDec {
	var tmp T
	t := reflect.TypeOf(tmp)
	params := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		params = append(params, t.Field(i).Name)
	}
	return FuncDec{name, params}
}

func Declare(name string, params []string) FuncDec {
	return FuncDec{name, params}
}
