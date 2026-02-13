package runtime

import (
	"sync"
	"testing"

	"GO_player/internal/memory/basegraph"
)

// Stress test: concurrent builds, reads, and cooldown/penalty updates on RuntimeGraph.
func TestRuntimeGraph_ConcurrentAccess(t *testing.T) {
	const (
		workers    = 16
		iterations = 1000
	)

	base := basegraph.NewBaseGraph()
	// Initialize base graph with some edges.
	for from := int64(1); from <= 5; from++ {
		for to := int64(10); to <= 15; to++ {
			base.Reinforce(from, to)
		}
	}

	rg := NewRuntimeGraph()
	rg.BuildFromBase(base)

	var wg sync.WaitGroup

	// Goroutines performing mixed read/write operations.
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				fromID := int64((j % 5) + 1)
				toID := int64(10 + (j % 6))

				switch j % 5 {
				case 0:
					_ = rg.GetEdges(fromID)
				case 1:
					rg.AddCooldown(fromID, toID, float64(j%3+1))
				case 2:
					rg.Penalty(fromID, toID)
				case 3:
					rg.ReduceCooldown()
				case 4:
					rg.RebuildFromBase(base, "concurrent rebuild")
				}
			}
		}(i)
	}

	wg.Wait()

	// Basic sanity check: GetEdges must not panic and must return a valid map.
	for from := int64(1); from <= 5; from++ {
		edges := rg.GetEdges(from)
		if edges == nil {
			t.Fatalf("GetEdges(%d) returned nil map", from)
		}
	}
}
