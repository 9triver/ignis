package system

import (
	"strconv"
	"strings"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type LinkLatencies struct {
	mu    sync.RWMutex
	links map[string]map[string]int64
}

func (lats *LinkLatencies) doSet(a1, a2 string, lat int64) {
	m, ok := lats.links[a1]
	if !ok {
		m = make(map[string]int64)
		lats.links[a1] = m
	}
	m[a2] = lat
}

func (lats *LinkLatencies) Set(a1, a2 string, lat int64) {
	if a1 == a2 {
		return
	}
	lats.mu.Lock()
	defer lats.mu.Unlock()

	lats.doSet(a1, a2, lat)
	lats.doSet(a2, a1, lat)
}

func (lats *LinkLatencies) Delete(addr string) {
	lats.mu.Lock()
	defer lats.mu.Unlock()

	delete(lats.links, addr)
	for _, m := range lats.links {
		delete(m, addr)
	}
}

func (lats *LinkLatencies) Of(p1 *actor.PIDSet, p2 *actor.PID) (int64, bool) {
	lats.mu.RLock()
	defer lats.mu.RUnlock()

	lat := int64(0)
	for _, pid := range p1.Values() {
		if pid.Address == p2.Address {
			continue
		}

		m, ok := lats.links[pid.Address]
		if !ok {
			return 0, false
		}
		l, ok := m[p2.Address]
		if !ok {
			return 0, false
		}
		lat += l
	}
	return lat, true
}

func (lats *LinkLatencies) String() string {
	lats.mu.RLock()
	defer lats.mu.RUnlock()

	sb := &strings.Builder{}
	for a1, m := range lats.links {
		for a2, lat := range m {
			sb.WriteString(a1)
			sb.WriteString(" -> ")
			sb.WriteString(a2)
			sb.WriteString(": ")
			sb.WriteString(strconv.Itoa(int(lat)))
			sb.WriteString("ms\n")
		}
	}

	return sb.String()
}

type ProcessLatency struct {
	mu          sync.RWMutex
	processCost map[*actor.PID]int64
	queuedTasks map[*actor.PID]int
}

func (lats *ProcessLatency) Inc(p *actor.PID) {
	if p == nil {
		return
	}

	lats.mu.Lock()
	defer lats.mu.Unlock()

	lats.queuedTasks[p]++
}

func (lats *ProcessLatency) Dec(p *actor.PID) {
	if p == nil {
		return
	}

	lats.mu.Lock()
	defer lats.mu.Unlock()

	tasks, ok := lats.queuedTasks[p]
	if !ok || tasks == 0 {
		return
	}
	lats.queuedTasks[p]--
}

func (lats *ProcessLatency) Set(p *actor.PID, lat int64) {
	lats.mu.Lock()
	defer lats.mu.Unlock()

	lats.processCost[p] = lat
}

func (lats *ProcessLatency) Delete(p *actor.PID) {
	lats.mu.Lock()
	defer lats.mu.Unlock()

	delete(lats.processCost, p)
}

func (lats *ProcessLatency) Of(p *actor.PID) (int64, bool) {
	lats.mu.RLock()
	defer lats.mu.RUnlock()

	lat, ok := lats.processCost[p]
	if !ok {
		return 0, false
	}

	tasks := lats.queuedTasks[p]
	return (lat + 1) * int64(tasks+1), true
}

func (lats *ProcessLatency) String() string {
	lats.mu.RLock()
	defer lats.mu.RUnlock()

	sb := &strings.Builder{}
	for p := range lats.processCost {
		sb.WriteString(p.Address + "/" + p.Id)
		sb.WriteString(": ")
		sb.WriteString(strconv.Itoa(int(lats.processCost[p])))
		sb.WriteString("ms, queued=")
		sb.WriteString(strconv.Itoa(lats.queuedTasks[p]))
		sb.WriteString("\n")
	}

	return sb.String()
}

type ActorLatencies struct {
	Process ProcessLatency
	Link    LinkLatencies
}

func (lats *ActorLatencies) Of(p1 *actor.PIDSet, p2 *actor.PID) (int64, bool) {
	p, ok := lats.Process.Of(p2)
	if !ok {
		return 0, false
	}

	l, ok := lats.Link.Of(p1, p2)
	if !ok {
		return 0, false
	}

	return p + l, true
}

func (lats *ActorLatencies) String() string {
	sb := &strings.Builder{}
	sb.WriteString("Process:\n")
	sb.WriteString(lats.Process.String())
	sb.WriteString("Link:\n")
	sb.WriteString(lats.Link.String())
	return sb.String()
}

func NewLatencies() *ActorLatencies {
	return &ActorLatencies{
		Process: ProcessLatency{
			processCost: make(map[*actor.PID]int64),
			queuedTasks: make(map[*actor.PID]int),
		},
		Link: LinkLatencies{
			links: make(map[string]map[string]int64),
		},
	}
}
