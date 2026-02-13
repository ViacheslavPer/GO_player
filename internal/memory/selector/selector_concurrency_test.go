package selector

import (
	"math"
	"sync"
	"testing"

	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
)

// Stress test: concurrent Selector.Next calls must not mutate RuntimeGraph probabilities.
func TestSelector_ImmutabilityUnderConcurrentNext(t *testing.T) {
	base := basegraph.NewBaseGraph()
	for i := int64(0); i < 10; i++ {
		base.Reinforce(1, 10)
		base.Reinforce(1, 20)
	}

	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(base)

	sel := NewSelector()

	edgesBefore := rg.GetEdges(1)
	sumBefore := 0.0
	for _, p := range edgesBefore {
		sumBefore += p
	}

	const (
		workers    = 16
		iterations = 1000
	)

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
		t.Fatalf("RuntimeGraph probabilities changed under concurrent Next: sum before %g, after %g", sumBefore, sumAfter)
	}
	if edgesAfter[10] != edgesBefore[10] || edgesAfter[20] != edgesBefore[20] {
		t.Fatalf("RuntimeGraph edge probabilities mutated under concurrent Next")
	}
}
