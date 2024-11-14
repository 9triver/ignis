package inference

import (
	"fmt"
	"os"
	"testing"

	tf "github.com/wamuir/graft/tensorflow"
	"github.com/wamuir/graft/tensorflow/op"
)

func TestInference(t *testing.T) {
	img, err := os.ReadFile("test.jpg")
	if err != nil {
		panic(err)
	}
	model, err := New("../saved_model")
	if err != nil {
		t.Fatal(err.Error())
	}
	res, err := Inference(model, img)
	if err != nil {
		t.Fatal(err.Error())
	}
	for i := range res.Classes {
		name, prob, box := res.Classes[i], res.Probabilities[i], res.Boxes[i]
		t.Logf("name: %s, prob: %f, box: %v", name, prob, box)
	}

	img, err = DrawBoxes(img, res)
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := os.WriteFile("output.jpg", img, 0644); err != nil {
		t.Fatal(err.Error())
	}
}

func TestTensorflow(t *testing.T) {
	s := op.NewScope()
	c := op.Const(s, "Hello from TensorFlow version "+tf.Version())
	graph, err := s.Finalize()
	if err != nil {
		panic(err)
	}

	// Execute the graph in a session.
	session, err := tf.NewSession(graph, nil)
	if err != nil {
		panic(err)
	}
	output, err := session.Run(nil, []tf.Output{c}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(output[0].Value())
}
