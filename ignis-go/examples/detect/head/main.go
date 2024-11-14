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

	system.MaximumMessageSize = 10 * 1024 * 1024
	m := ipc.GetVenvManager()
	defer m.Close()
	m.Run()

	capturePkl, err := os.ReadFile("detect/capture.pkl")
	if err != nil {
		panic(err)
	}

	modelPkl, err := os.ReadFile("detect/detect.pkl")
	if err != nil {
		panic(err)
	}

	paintPkl, err := os.ReadFile("detect/paint.pkl")
	if err != nil {
		panic(err)
	}

	capture, err := handlers.NewPyFunc("capture", []string{"device"}, "detect", []string{}, capturePkl)
	if err != nil {
		panic(err)
	}

	detect, err := handlers.NewPyFunc("detect", []string{"im"}, "detect", []string{}, modelPkl)
	if err != nil {
		panic(err)
	}

	paint, err := handlers.NewPyFunc("paint", []string{"im", "pred"}, "detect", []string{}, paintPkl)
	if err != nil {
		panic(err)
	}

	system.Spawn(compute.NewActor(capture, system.LatencyNotSpecified).Props())
	system.Spawn(compute.NewActor(detect, system.LatencyNotSpecified).Props())
	system.Spawn(compute.NewActor(paint, system.LatencyNotSpecified).Props())

	_, _ = console.ReadLine()
}
