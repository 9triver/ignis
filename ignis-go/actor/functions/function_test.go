package functions

import (
	"testing"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
)

func TestStreamJoinFunc(t *testing.T) {
	type I struct {
		Ints <-chan any
	}

	type O struct {
		Sum int
	}

	innerFunc := func(input I) (O, error) {
		sum := 0
		for i := range input.Ints {
			sum += i.(int)
		}
		return O{Sum: sum}, nil
	}

	inputs := make(chan int)
	go func() {
		defer close(inputs)
		for i := range 10 {
			inputs <- i
		}
	}()

	f := NewGo("sum", innerFunc, proto.LangGo)
	s := messages.NewLocalStream(inputs, proto.LangGo)
	invoke := map[string]messages.Object{}
	invoke["Ints"] = s
	r, err := f.Call(invoke)
	if err != nil {
		t.Fatal(errors.Stacktrace(err))
	}
	t.Log(r)
}

func TestStreamStreamFunc(t *testing.T) {
	type N struct {
		Num int
	}

	type I struct {
		Ints <-chan int
	}

	type O struct {
		Sum int
	}

	generateInts := func(input N) (<-chan int, error) {
		ch := make(chan int)
		go func() {
			defer close(ch)
			for i := range input.Num {
				ch <- i
			}
		}()

		return ch, nil
	}

	getSum := func(input I) (O, error) {
		sum := 0
		for i := range input.Ints {
			println(i)
			sum += i
		}
		return O{Sum: sum}, nil
	}

	f1 := NewGo("genInts", generateInts, proto.LangGo)
	invoke := map[string]messages.Object{}
	invoke["Num"] = messages.NewLocalObject(10, proto.LangGo)
	r1, err := f1.Call(invoke)
	if err != nil {
		t.Fatal(errors.Stacktrace(err))
	}

	f2 := NewGo("getSum", getSum, proto.LangGo)
	invoke2 := map[string]messages.Object{}
	invoke2["Ints"] = r1
	r2, err := f2.Call(invoke2)
	if err != nil {
		t.Fatal(errors.Stacktrace(err))
	}

	v, _ := r2.GetValue()
	t.Log(v)
}
