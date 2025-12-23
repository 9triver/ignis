package unikernel

import (
	_ "embed"
	"testing"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/remote"
	"github.com/9triver/ignis/object"
)

var (
	//go:embed "handlers.ml"
	handlers string
)

func TestMirage(t *testing.T) {
	manager := remote.NewManager("0.0.0.0", 8085)

	f, err := functions.NewUnikernel(manager, "add", []string{"a", "b"}, handlers)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := f.Call(map[string]object.Interface{
		"a": object.NewLocal(20, object.LangJson),
		"b": object.NewLocal(20, object.LangJson),
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(obj, 111)
}
