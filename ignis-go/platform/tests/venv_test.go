package platform_test

import (
	"actors/platform/actor/head"
	"actors/platform/handlers/ipc"
	"actors/platform/store"
	"actors/proto"
	"os"
	"testing"
	"time"
)

func TestVenv(t *testing.T) {
	head.Start("127.0.0.1", 2312)

	m := ipc.GetVenvManager()
	defer m.Close()
	m.Run()

	v, err := m.GetVenv("test")
	if err != nil {
		t.Fatal(err)
	}

	pkl, err := os.ReadFile("func.pkl")
	if err != nil {
		t.Fatal(err)
	}

	calcHandler := ipc.MakeAddHandler("calc", pkl, store.LangJson, []string{"add"})
	v.Tell(ipc.MakeRouterMessage(calcHandler))

	s := store.New()
	params := map[string]*store.Object{
		"a": s.Add(1, store.LangJson),
		"b": s.Add(2, store.LangJson),
	}
	encParams := make(map[string]*proto.EncodedObject)
	for k, v := range params {
		encParams[k], _ = v.Encode()
	}

	tic := time.Now()
	exec := ipc.MakeExecute("calc", "add", encParams)

	ret, err := v.Execute(exec)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("python ipc", time.Since(tic), ret.Value)
}
