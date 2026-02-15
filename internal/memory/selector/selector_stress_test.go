package selector

import (
	"math"
	"sync"
	"testing"

	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
)

// TestSelector_ImmutabilityHighParallelism verifies Selector.Next does not mutate RuntimeGraph with >100 goroutines.
func TestSelector_ImmutabilityHighParallelism(t *testing.T) {
	base := basegraph.NewBaseGraph()
	for i := int64(0); i < 15; i++ {
		base.Reinforce(1, 10)
		base.Reinforce(1, 20)
		base.Reinforce(1, 30)
	}
	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(base)

	sel := NewSelector()
	edgesBefore := rg.GetEdges(1)
	sumBefore := 0.0
	for _, p := range edgesBefore {
		sumBefore += p
	}

	const workers = 120
	const iterations = 500
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sel.Next(1, rg)
			}
		}()
	}
	wg.Wait()

	edgesAfter := rg.GetEdges(1)
	sumAfter := 0.0
	for _, p := range edgesAfter {
		sumAfter += p
	}
	if math.Abs(sumBefore-1.0) > 1e-9 || math.Abs(sumAfter-1.0) > 1e-9 {
		t.Fatalf("probabilities sum changed: before %g, after %g", sumBefore, sumAfter)
	}
	if edgesAfter[10] != edgesBefore[10] || edgesAfter[20] != edgesBefore[20] || edgesAfter[30] != edgesBefore[30] {
		t.Fatal("RuntimeGraph edge probabilities mutated by concurrent Selector.Next")
	}
}

// TestSelector_ConcurrentNextFromMultipleFromIDs exercises Next with different fromIDs concurrently.
func TestSelector_ConcurrentNextFromMultipleFromIDs(t *testing.T) {
	base := basegraph.NewBaseGraph()
	for from := int64(0); from <= 5; from++ {
		for to := int64(10); to <= 15; to++ {
			base.Reinforce(from, to)
		}
	}
	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(base)
	sel := NewSelector()

	var wg sync.WaitGroup
	wg.Add(80)
	for i := 0; i < 80; i++ {
		go func(id int) {
			defer wg.Done()
			fromID := int64(id % 6)
			for j := 0; j < 200; j++ {
				_, _ = sel.Next(fromID, rg)
			}
		}(i)
	}
	wg.Wait()
}

// TestSelector_EmptyGraphConcurrentNext verifies Next(unknownFromID, rg) with empty edges does not panic.
func TestSelector_EmptyGraphConcurrentNext(t *testing.T) {
	base := basegraph.NewBaseGraph()
	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(base)
	sel := NewSelector()

	var wg sync.WaitGroup
	var errCount int
	var errMu sync.Mutex
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				toID, ok := sel.Next(99, rg)
				if ok && toID != 0 {
					errMu.Lock()
					errCount++
					errMu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	if errCount > 0 {
		t.Errorf("Next(99, empty rg) returned non-zero toID %d times", errCount)
	}
}
