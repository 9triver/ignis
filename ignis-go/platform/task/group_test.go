package task

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	g := NewGroup("test")
	go g.Run()

	go func() {
		for i := range 15 {
			if i > 8 {
				<-time.After(time.Duration(rand.IntN(3000)) * time.Millisecond)
			}
			g.Push(&ActorInfo{
				LinkLatency: rand.Int64N(2000),
				CalcLatency: rand.Int64N(10000),
			})
		}
	}()

	for range 15 {
		t.Log("Current: ", g.pq.String())
		selected := <-g.SelectChan()
		t.Log("Select: ", selected)
		<-time.After(time.Duration(rand.IntN(1000)) * time.Millisecond)
	}
}
