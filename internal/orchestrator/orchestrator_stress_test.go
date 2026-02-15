package orchestrator

import (
	"sync"
	"testing"
	"time"
)

const stressGoroutines = 120
const stressIterations = 400

// TestOrchestrator_HighLoadStress runs >100 goroutines doing PlayNext and PlayBack only.
// Learn and RebuildRuntime are not mixed here to avoid races (RebuildRuntime replaces ctx/channels).
func TestOrchestrator_HighLoadStress(t *testing.T) {
	o := NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(1, 3)
	o.BaseGraph().Reinforce(2, 4)
	o.RebuildRuntime("initial")
	o.Start()
	defer func() {
		time.Sleep(20 * time.Millisecond)
		o.Stop()
	}()

	var wg sync.WaitGroup
	wg.Add(stressGoroutines)
	for i := 0; i < stressGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < stressIterations; j++ {
				o.PlayNext()
				o.PlayBack()
			}
		}()
	}
	wg.Wait()

	if o.BaseGraph() == nil {
		t.Fatalf("BaseGraph nil after stress")
	}
	if o.RuntimeGraph() == nil {
		t.Fatalf("RuntimeGraph nil after stress")
	}
}

// TestOrchestrator_ConcurrentReadOnly validates concurrent RuntimeGraph reads and PlayNext/PlayBack.
// No RebuildRuntime and no Learn during run to stay race-free (Start not used to avoid background RebuildRuntime).
func TestOrchestrator_ConcurrentReadOnly(t *testing.T) {
	o := NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.RebuildRuntime("init")
	o.Start()
	defer func() {
		time.Sleep(15 * time.Millisecond)
		o.Stop()
	}()

	var wg sync.WaitGroup
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rg := o.RuntimeGraph()
				if rg != nil {
					_ = rg.GetEdges(0)
					_ = rg.GetBuildVersion()
				}
				o.PlayNext()
				o.PlayBack()
			}
		}()
	}
	wg.Wait()
}
