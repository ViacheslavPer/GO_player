package orchestrator

import (
	"sync"
	"testing"
	"time"
)

// Stress test: concurrent Learn / playback / rebuild interactions.
func TestOrchestrator_ConcurrentUsage(t *testing.T) {
	const (
		workers    = 16
		iterations = 1000
	)

	o := NewOrchestrator()
	o.Start()
	defer func() {
		// Give background goroutines a brief moment to drain channels before stopping.
		time.Sleep(10 * time.Millisecond)
		o.Stop()
	}()

	var wg sync.WaitGroup

	// Workers performing Learn operations.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < workers*iterations; i++ {
			fromID := int64((i % 5) + 1)
			toID := int64(10 + (i % 6))
			o.Learn(fromID, toID)
		}
	}()

	// Workers performing playback navigation.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < workers*iterations; i++ {
			if i%2 == 0 {
				o.PlayNext()
			} else {
				o.PlayBack()
			}
		}
	}()

	// Worker periodically triggering explicit rebuilds in parallel with background ones.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			o.RebuildRuntime("explicit rebuild in stress test")
		}
	}()

	// Worker to poke diff-based rebuild path via internal channel.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			lc := o.lifecycle.Load()
			if lc == nil {
				return
			}
			select {
			case o.diffChan <- struct{}{}:
			case <-lc.ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	// Sanity checks: internal pointers should remain non-nil.
	if o.BaseGraph() == nil {
		t.Fatalf("BaseGraph is nil after concurrent usage")
	}
	if o.RuntimeGraph() == nil {
		t.Fatalf("RuntimeGraph is nil after concurrent usage")
	}
}
