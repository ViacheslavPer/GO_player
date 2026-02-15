package runtime

import (
	"sync"
	"testing"

	"GO_player/internal/memory/basegraph"
)

const stressWorkers = 120
const stressIterations = 500

// TestRuntimeGraph_HighLoadStress runs >100 goroutines with mixed read/write operations.
func TestRuntimeGraph_HighLoadStress(t *testing.T) {
	base := basegraph.NewBaseGraph()
	for from := int64(0); from <= 10; from++ {
		for to := int64(20); to <= 30; to++ {
			base.Reinforce(from, to)
		}
	}

	rg := NewRuntimeGraph()
	rg.BuildFromBase(base)

	var wg sync.WaitGroup
	wg.Add(stressWorkers)
	for i := 0; i < stressWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < stressIterations; j++ {
				fromID := int64((id + j) % 11)
				toID := int64(20 + (j % 11))
				switch j % 6 {
				case 0:
					_ = rg.GetEdges(fromID)
				case 1:
					_ = rg.GetBuildVersion()
				case 2:
					rg.AddCooldown(fromID, toID, float64(j%5+1))
				case 3:
					rg.Penalty(fromID, toID)
				case 4:
					rg.ReduceCooldown()
				case 5:
					rg.Reinforce(fromID, toID)
				}
			}
		}(i)
	}
	wg.Wait()

	for from := int64(0); from <= 10; from++ {
		edges := rg.GetEdges(from)
		if edges == nil {
			t.Fatalf("GetEdges(%d) returned nil after stress", from)
		}
	}
}

// TestRuntimeGraph_RapidRebuildLoop stresses RebuildFromBase and CopyBase while others read.
func TestRuntimeGraph_RapidRebuildLoop(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(0, 1)
	base.Reinforce(1, 2)

	rg := NewRuntimeGraph()
	rg.BuildFromBase(base)

	var wg sync.WaitGroup
	const rebuilders = 20
	const readers = 50
	const loops = 100

	wg.Add(rebuilders)
	for i := 0; i < rebuilders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				rg.RebuildFromBase(base, "rapid rebuild")
			}
		}()
	}

	wg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops*2; j++ {
				_ = rg.GetEdges(0)
				_ = rg.GetEdges(1)
				_ = rg.GetBuildVersion()
				_ = rg.GetTimestamp()
			}
		}()
	}

	wg.Wait()
}

// TestRuntimeGraph_ConcurrentCooldownFlood stresses AddCooldown and ReduceCooldown.
func TestRuntimeGraph_ConcurrentCooldownFlood(t *testing.T) {
	base := basegraph.NewBaseGraph()
	for from := int64(1); from <= 5; from++ {
		for to := int64(10); to <= 15; to++ {
			base.Reinforce(from, to)
		}
	}
	rg := NewRuntimeGraph()
	rg.BuildFromBase(base)

	var wg sync.WaitGroup
	const workers = 80
	const iters = 300

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				fromID := int64((id % 5) + 1)
				toID := int64(10 + (j % 6))
				if j%2 == 0 {
					rg.AddCooldown(fromID, toID, float64(j%10+1))
				} else {
					rg.ReduceCooldown()
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestRuntimeGraph_EmptyGraphHeavyReads verifies empty graph under concurrent GetEdges.
func TestRuntimeGraph_EmptyGraphHeavyReads(t *testing.T) {
	rg := NewRuntimeGraph()
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = rg.GetEdges(int64(j))
				_ = rg.GetBuildVersion()
				_ = rg.GetDiffts()
			}
		}()
	}
	wg.Wait()
}
