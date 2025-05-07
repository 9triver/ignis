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
		for i := range 10 {
			if i > 5 {
				<-time.After(time.Duration(rand.IntN(3)) * time.Second)
			}
			g.Push(&ActorInfo{LinkLatency: rand.Int64N(10) * int64(time.Second)})
			t.Log("Current: ", g.pq.String())
		}
	}()

	for selected := range g.SelectChan() {
		<-time.After(time.Duration(rand.IntN(3)) * time.Second)
		t.Log("Select: ", selected)
	}
}
