package benchmarks

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
)

func TestSchd(t *testing.T) {
	defer SaveResult("schd")

	cloudLink := 200 * time.Millisecond
	cloudLat := 100 * time.Millisecond
	edgeLink := 20 * time.Millisecond
	edgeLat := 600 * time.Millisecond

	workerFunc := func(a, b int, lat time.Duration) int {
		time.Sleep(lat)
		return a + b
	}

	doTest := func(nClouds, nEdges, nTasks int) {
		g := task.NewGroup("test")

		for range nClouds {
			g.Push(&task.ActorInfo{
				CalcLatency: cloudLat.Milliseconds(),
				LinkLatency: cloudLink.Milliseconds(),
				Ref: &proto.ActorRef{
					ID: "cloud",
				},
			})
		}

		for range nEdges {
			g.Push(&task.ActorInfo{
				CalcLatency: edgeLat.Milliseconds(),
				LinkLatency: edgeLink.Milliseconds(),
				Ref: &proto.ActorRef{
					ID: "edge",
				},
			})
		}

		var totalLatency time.Duration
		var wg sync.WaitGroup
		wg.Add(nTasks)

		tic := time.Now()
		var cloud, edge atomic.Int32

		cloudStats := make([]int, nTasks)
		edgeStats := make([]int, nTasks)
		for i := range nTasks {
			selected := g.Select()
			isCloud := selected.Ref.ID == "cloud"
			go func() {
				time.Sleep(time.Duration(selected.LinkLatency) * time.Millisecond)
				workerFunc(2, 2, time.Duration(selected.CalcLatency)*time.Millisecond)
				time.Sleep(time.Duration(selected.LinkLatency) * time.Millisecond)
				wg.Done()

				g.Push(selected)
				if isCloud {
					cloud.Add(-1)
				} else {
					edge.Add(-1)
				}
				t.Logf("task done from %s. Cloud: %d, Edge: %d", selected.Ref.ID, cloud.Load(), edge.Load())
			}()

			if isCloud {
				cloud.Add(1)
			} else {
				edge.Add(1)
			}

			cloudStats[i] = int(cloud.Load())
			edgeStats[i] = int(edge.Load())
			t.Logf("new task assigned to %s. Cloud: %d, Edge: %d", selected.Ref.ID, cloud.Load(), edge.Load())
		}
		wg.Wait()
		totalLatency = time.Since(tic)

		average := totalLatency / time.Duration(nTasks)

		t.Logf("Average latency: %v", average)
		WriteResult(
			"nClouds", nClouds,
			"nEdges", nEdges,
			"nTasks", nTasks,
			"cloudStats", cloudStats,
			"edgeStats", edgeStats,
		)
	}

	nClouds := 2
	nEdges := 10
	nTasks := 256

	doTest(nClouds, nEdges, nTasks)
}

func TestSchdOverhead(t *testing.T) {
	defer SaveResult("schd_overhead")

	workerFunc := func(a, b int, lat time.Duration) int {
		time.Sleep(lat)
		return a + b
	}

	doTest := func(edgeLink, edgeLat time.Duration, nTasks, nEdges int) {
		t.Logf("Schd overhead bench for %d workers and %d tasks", nEdges, nTasks)
		g := task.NewGroup("test")
		for range nEdges {
			g.Push(&task.ActorInfo{
				CalcLatency: edgeLat.Milliseconds(),
				LinkLatency: edgeLink.Milliseconds(),
				Ref: &proto.ActorRef{
					ID: "edge",
				},
			})
		}

		var totalLatency time.Duration
		var wg sync.WaitGroup
		wg.Add(nTasks)

		tic := time.Now()
		for range nTasks {
			selected := g.Select()
			go func() {
				time.Sleep(time.Duration(selected.LinkLatency) * time.Millisecond)
				workerFunc(2, 2, time.Duration(selected.CalcLatency)*time.Millisecond)
				time.Sleep(time.Duration(selected.LinkLatency) * time.Millisecond)
				wg.Done()

				g.Push(selected)
			}()
		}

		wg.Wait()
		totalLatency = time.Since(tic)

		average := totalLatency / time.Duration(nTasks)
		expected := (edgeLink*2 + edgeLat) / time.Duration(nEdges)
		overhead := average - expected

		t.Logf("Average latency: %v, theoretically %v", average, expected)
		t.Logf("Average overhead: %v (%f%%)", overhead, 100*float64(average)/float64(expected))
		t.Log("============================")

		WriteResult(
			"link_us", edgeLink.Microseconds(),
			"compute_us", edgeLat.Microseconds(),
			"nTasks", nTasks,
			"nEdges", nEdges,
			"latency_us", average.Microseconds(),
			"overhead_us", overhead.Microseconds(),
			"overhead_per", 100*float64(average)/float64(expected),
		)
	}

	edgeLink := 100 * time.Millisecond
	edgeLat := 100 * time.Millisecond

	doTest(edgeLink, edgeLat, 100, 5)
	doTest(edgeLink, edgeLat, 100, 10)
	doTest(edgeLink, edgeLat, 100, 50)
	doTest(edgeLink, edgeLat, 100, 100)

	time.Sleep(3 * time.Second)

	doTest(edgeLink, edgeLat, 200, 5)
	doTest(edgeLink, edgeLat, 200, 10)
	doTest(edgeLink, edgeLat, 200, 50)
	doTest(edgeLink, edgeLat, 200, 100)

	time.Sleep(3 * time.Second)

	doTest(edgeLink, edgeLat, 500, 5)
	doTest(edgeLink, edgeLat, 500, 10)
	doTest(edgeLink, edgeLat, 500, 50)
	doTest(edgeLink, edgeLat, 500, 100)
}
