package main

import (
	"actors/example/detect-go/camera"
	"actors/example/detect-go/inference"
	"actors/platform/actor/compute"
	"actors/platform/cmd/server"
	"actors/platform/handlers"
	"actors/platform/store"
	"actors/platform/system"
	"fmt"
	"time"
)

func headMain(config *server.Config) error {
	sys := system.ActorSystem()
	logger := sys.Logger()

	cam := camera.New()
	model, err := inference.New("saved_model")
	if err != nil {
		return err
	}

	inf := handlers.DeclareTyped[struct{ Image []byte }, *inference.Result]("inference")
	impl := handlers.ImplGo(inf, func(req *struct{ Image []byte }) (*inference.Result, error) {
		logger.Info("inference on head")
		return inference.Inference(model, req.Image)
	}, store.LangJson)

	capture := handlers.NewGo("capture", func(req *struct {
		Device string
	}) ([]byte, error) {
		logger.Info("capture", "req", req)
		if err := cam.Reset(req.Device); err != nil {
			return nil, fmt.Errorf("reset: %w", err)
		}
		return cam.Capture(10 * time.Second)
	}, store.LangJson)

	draw := handlers.NewGo("draw", func(req *struct {
		Image  []byte
		Result map[string]any
	}) ([]byte, error) {
		logger.Info("draw")
		return inference.DrawBoxesJson(req.Image, req.Result)
	}, store.LangJson)

	sys.Root.Spawn(compute.NewActor(impl, system.LatencyNotSpecified).Props())
	sys.Root.Spawn(compute.NewActor(capture, system.LatencyNotSpecified).Props())
	sys.Root.Spawn(compute.NewActor(draw, system.LatencyNotSpecified).Props())
	// s := store.New()
	// _, _ = console.ReadLine()

	// manager := system.HeadRef().DAGManager()
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.Create{}))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendNode{
	// 	Name:   "capture",
	// 	Params: capture.Params(),
	// 	Option: &protoDAG.AppendNode_Actor{Actor: capturePID},
	// }))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendNode{
	// 	Name:   "inference",
	// 	Params: inf.Params(),
	// 	Option: &protoDAG.AppendNode_Group{Group: "inference"},
	// }))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendNode{
	// 	Name:   "draw",
	// 	Params: draw.Params(),
	// 	Option: &protoDAG.AppendNode_Actor{Actor: drawPID},
	// }))
	// obj, _ := s.Add("/dev/video0", store.LangJson).Encode()
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendNode{
	// 	Name:   "driver",
	// 	Option: &protoDAG.AppendNode_Value{Value: obj},
	// }))

	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
	// 	From:  "driver",
	// 	To:    "capture",
	// 	Param: "Device",
	// }))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
	// 	From:  "capture",
	// 	To:    "inference",
	// 	Param: "Image",
	// }))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
	// 	From:  "capture",
	// 	To:    "draw",
	// 	Param: "Image",
	// }))
	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
	// 	From:  "inference",
	// 	To:    "draw",
	// 	Param: "Result",
	// }))

	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.AppendOutput{
	// 	Node: "draw",
	// }))

	// sys.Root.Send(manager, dag.MakeCommand("test-dag", &protoDAG.Serve{}))
	// resp, err := sys.Root.RequestFuture(manager, dag.MakeCommand("test-dag", &protoDAG.Execute{}), 1000*time.Second).Result()
	// if err != nil {
	// 	return err
	// }
	// if res, ok := resp.(*protoDAG.ExecutionResult); ok {
	// 	logger.Info("result got")
	// 	img := res.Results["draw"]
	// 	logger.Info("img", "len", len(img.Data))
	// }

	return nil
}

func main() {
	system.MaximumMessageSize = 10 * 1024 * 1024
	server.Execute(headMain)
}
