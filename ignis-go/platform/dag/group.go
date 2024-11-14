package dag

import (
	"actors/platform/system"
	"math"
	"math/rand"

	"github.com/asynkron/protoactor-go/actor"
)

type ActorGroup struct {
	candidates *actor.PIDSet
	latencies  *system.ActorLatencies
}

func (ag *ActorGroup) SelectRandom() *actor.PID {
	l := ag.candidates.Len()
	r := rand.Intn(l)
	return ag.candidates.Get(r)
}

func (ag *ActorGroup) Select(sources *actor.PIDSet) (candidate *actor.PID) {
	lat := int64(math.MaxInt64)
	for _, pid := range ag.candidates.Values() {
		if l, ok := ag.latencies.Of(sources, pid); ok && l < lat {
			lat = l
			candidate = pid
		}
	}

	return candidate
}

func (ag *ActorGroup) TaskStart(candidate *actor.PID) {
	ag.latencies.Process.Inc(candidate)
}

func (ag *ActorGroup) TaskDone(candidate *actor.PID) {
	ag.latencies.Process.Dec(candidate)
}

func NewActorGroup(candidates *actor.PIDSet) *ActorGroup {
	return &ActorGroup{
		candidates: candidates,
		latencies:  system.HeadRef().Latencies(),
	}
}
