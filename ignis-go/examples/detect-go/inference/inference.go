package inference

import (
	"image"
	"image/color"

	"github.com/mitchellh/mapstructure"
	tf "github.com/wamuir/graft/tensorflow"
	"github.com/wamuir/graft/tensorflow/op"
	"gocv.io/x/gocv"
)

func New(path string) (*tf.SavedModel, error) {
	model, err := tf.LoadSavedModel(path, []string{"serve"}, &tf.SessionOptions{})

	if err != nil {
		return nil, err
	}
	return model, nil
}

func decodeImg(img []byte) (*tf.Tensor, error) {
	scope := op.NewScope()
	tensor, err := tf.NewTensor(string(img))
	if err != nil {
		return nil, err
	}

	output := op.ExpandDims(
		scope,
		op.DecodeJpeg(scope, op.Const(scope, tensor), op.DecodeJpegChannels(3)),
		op.Const(scope.SubScope("make_batch"), int32(0)),
	)

	graph, err := scope.Finalize()
	if err != nil {
		return nil, err
	}

	session, err := tf.NewSession(graph, &tf.SessionOptions{})
	if err != nil {
		return nil, err
	}

	normalized, err := session.Run(nil, []tf.Output{output}, nil)
	if err != nil {
		return nil, err
	}

	return normalized[0], nil
}

type Result struct {
	Boxes         []struct{ x1, y1, x2, y2 int } `json:"boxes"`
	Classes       []string                       `json:"classes"`
	Probabilities []float64                      `json:"probabilities"`
}

func decodeResult(
	w, h int64,
	boxes [][]float32,
	probabilities []float32,
	classes []float32,
) *Result {
	results := new(Result)

	iw, ih := float64(w), float64(h)
	for i, p := range probabilities {
		if p > 0.5 {
			// Box coordinates come in as [y1, x1, y2, x2]
			x1 := int(float64(boxes[i][1]) * iw)
			x2 := int(float64(boxes[i][3]) * iw)
			y1 := int(float64(boxes[i][0]) * ih)
			y2 := int(float64(boxes[i][2]) * ih)

			results.Boxes = append(results.Boxes, struct{ x1, y1, x2, y2 int }{x1, y1, x2, y2})
			results.Probabilities = append(results.Probabilities, float64(p))
			results.Classes = append(results.Classes, COCO_SSD_LABELS[int(classes[i])])
		}
	}
	return results
}

func DrawBoxesJson(img []byte, resultsJson map[string]any) ([]byte, error) {
	results := &Result{}
	if err := mapstructure.Decode(resultsJson, &results); err != nil {
		return nil, err
	}
	return DrawBoxes(img, results)
}

func DrawBoxes(img []byte, results *Result) ([]byte, error) {
	mat, err := gocv.IMDecode(img, gocv.IMReadColor)
	if err != nil {
		return nil, err
	}

	for i := range results.Boxes {
		x1, y1, x2, y2 := results.Boxes[i].x1, results.Boxes[i].y1, results.Boxes[i].x2, results.Boxes[i].y2
		gocv.Rectangle(&mat, image.Rect(x1, y1, x2, y2), color.RGBA{0, 255, 0, 0}, 4)
		gocv.PutText(&mat, results.Classes[i], image.Pt(x1+6, y1-6), gocv.FontHersheyPlain, 1.5, color.RGBA{0, 255, 0, 0}, 4)
	}

	buf, err := gocv.IMEncode(gocv.PNGFileExt, mat)
	if err != nil {
		return nil, err
	}
	return buf.GetBytes(), nil
}

func InferenceJson(model *tf.SavedModel, img []byte) (map[string]any, error) {
	results, err := Inference(model, img)
	if err != nil {
		return nil, err
	}
	return map[string]any{"boxes": results.Boxes, "classes": results.Classes, "probabilities": results.Probabilities}, nil
}

func Inference(model *tf.SavedModel, img []byte) (*Result, error) {
	graph, session := model.Graph, model.Session

	i := graph.Operation("image_tensor")
	o1 := graph.Operation("detection_boxes")
	o2 := graph.Operation("detection_scores")
	o3 := graph.Operation("detection_classes")
	o4 := graph.Operation("num_detections")

	tensor, err := decodeImg(img)
	if err != nil {
		return nil, err
	}
	shape := tensor.Shape()
	h, w := shape[1], shape[2]

	output, err := session.Run(
		map[tf.Output]*tf.Tensor{i.Output(0): tensor},
		[]tf.Output{o1.Output(0), o2.Output(0), o3.Output(0), o4.Output(0)},
		nil,
	)
	if err != nil {
		return nil, err
	}

	boxes := output[0].Value().([][][]float32)[0]
	probabilities := output[1].Value().([][]float32)[0]
	classes := output[2].Value().([][]float32)[0]
	return decodeResult(w, h, boxes, probabilities, classes), nil
}
