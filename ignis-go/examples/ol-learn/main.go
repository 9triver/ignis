package main

import (
	"actors/platform/actor/compute"
	"actors/platform/actor/head"
	"actors/platform/actor/http"
	"actors/platform/handlers"
	"actors/platform/handlers/ipc"
	"actors/platform/system"
	"os"

	console "github.com/asynkron/goconsole"
)

func main() {
	head.Start("127.0.0.1", 2312)
	http.Start(2313)

	m := ipc.GetVenvManager()
	defer m.Close()
	m.Run()

	collectPkl, err := os.ReadFile("collect.pkl")
	if err != nil {
		panic(err)
	}

	modelPkl, err := os.ReadFile("river.pkl")
	if err != nil {
		panic(err)
	}

	collect, err := handlers.NewPyFunc("collect", []string{"train"}, "test", []string{}, collectPkl)
	if err != nil {
		panic(err)
	}

	model, err := handlers.NewPyInstance(
		"Model", map[string][]string{
			"update":  {"info"},
			"predict": {"info"},
		}, "test", []string{}, modelPkl,
	)

	if err != nil {
		panic(err)
	}

	for _, method := range model.Functions() {
		system.Spawn(compute.NewActor(method, system.LatencyNotSpecified).Props())
	}
	system.Spawn(compute.NewActor(collect, system.LatencyNotSpecified).Props())

	_, _ = console.ReadLine()
}
