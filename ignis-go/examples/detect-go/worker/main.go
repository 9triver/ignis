package main

import (
	"actors/example/detect-go/inference"
	"actors/platform/actor/compute"
	"actors/platform/cmd/server"
	"actors/platform/handlers"
	"actors/platform/store"
	"actors/platform/system"
)

func main() {
	system.MaximumMessageSize = 10 * 1024 * 1024

	server.Execute(func(config *server.Config) error {
		logger := system.Logger()
		model, err := inference.New("saved_model")
		if err != nil {
			panic(err)
		}

		inf := handlers.DeclareTyped[struct{ Image []byte }, map[string]any]("inference")
		impl := handlers.ImplGo(inf, func(req *struct{ Image []byte }) (map[string]any, error) {
			logger.Info("inference on worker")
			return inference.InferenceJson(model, req.Image)
		}, store.LangJson)

		system.Spawn(compute.NewActor(impl, system.LatencyNotSpecified).Props())
		return nil
	})
}
