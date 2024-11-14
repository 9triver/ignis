package main

import (
	"actors/platform/actor/compute"
	"actors/platform/actor/worker"
	"actors/platform/handlers"
	"actors/platform/handlers/ipc"
	"actors/platform/system"
	"os"

	console "github.com/asynkron/goconsole"
)

func main() {
	system.MaximumMessageSize = 10 * 1024 * 1024
	worker.Start("127.0.0.1", "127.0.0.1", 2312)
	m := ipc.GetVenvManager()
	defer m.Close()
	m.Run()

	modelPkl, err := os.ReadFile("detect/detect.pkl")
	if err != nil {
		panic(err)
	}

	detect, err := handlers.NewPyFunc("detect", []string{"im"}, "detect", []string{}, modelPkl)
	if err != nil {
		panic(err)
	}

	system.Spawn(compute.NewActor(detect, system.LatencyNotSpecified).Props())
	_, _ = console.ReadLine()
}
