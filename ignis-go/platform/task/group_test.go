package task

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	g := NewGroup("test")

	go func() {
		for i := range 12 {
			if i > 8 {
				<-time.After(time.Duration(rand.IntN(3000)) * time.Millisecond)
			}
			g.Push(&ActorInfo{
				LinkLatency: rand.Int64N(2000),
				CalcLatency: rand.Int64N(10000),
			})
		}
	}()

	for range 12 {
		selected := g.Select()
		t.Log("Select: ", selected)
		<-time.After(time.Duration(rand.IntN(1000)) * time.Millisecond)
	}
}
