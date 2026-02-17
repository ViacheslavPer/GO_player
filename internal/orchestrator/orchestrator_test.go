package orchestrator

import (
	"sync"
	"testing"
	"time"
)

func TestOrchestrator_NewOrchestrator_StopIsSafe(t *testing.T) {
	o := NewOrchestrator()
	if o == nil {
		t.Fatalf("NewOrchestrator() returned nil")
	}
	if o.BaseGraph() == nil {
		t.Fatalf("BaseGraph() returned nil")
	}
	if o.RuntimeGraph() == nil {
		t.Fatalf("RuntimeGraph() returned nil")
	}
	if o.PlaybackChain() == nil {
		t.Fatalf("PlaybackChain() returned nil")
	}

	// Stop before Start should not deadlock.
	done := make(chan struct{})
	go func() {
		defer close(done)
		o.Stop()
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Stop() timed out (possible deadlock)")
	}
}

func TestOrchestrator_StartStop_Completes(t *testing.T) {
	o := NewOrchestrator()
	o.Start()

	done := make(chan struct{})
	go func() {
		defer close(done)
		o.Stop()
	}()
	select {
	case <-done:
	case <-time.After(800 * time.Millisecond):
		t.Fatalf("Start/Stop timed out (goroutines may not exit)")
	}
}

// ProcessFeedbak sends a signal on a bounded channel.
// If Start() is never called, sustained feedback can apply backpressure (blocking).
// This test validates that Stop() cancels the lifecycle and unblocks a potentially
// blocked sender (i.e., no permanent deadlock).
func TestOrchestrator_ProcessFeedbak_BlockingIsCancelableWithoutStart(t *testing.T) {
	o := NewOrchestrator()

	// Fill the buffered channel.
	for i := 0; i < cap(o.diffChan); i++ {
		o.ProcessFeedbak(1, 2, 1.0, 10.0)
	}

	// Next call may block due to backpressure; ensure Stop() can always unblock it.
	done := make(chan struct{})
	go func() {
		defer close(done)
		o.ProcessFeedbak(1, 2, 1.0, 10.0)
	}()

	time.Sleep(25 * time.Millisecond) // allow the goroutine to potentially block
	o.Stop()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("ProcessFeedbak did not exit after Stop() (possible deadlock)")
	}
}

func TestOrchestrator_ConcurrentUse_StressRace(t *testing.T) {
	o := NewOrchestrator()
	o.Start()
	defer o.Stop()

	const (
		workers   = 32
		opsPerWkr = 2000
	)

	var wg sync.WaitGroup
	wg.Add(workers)

	start := time.Now()
	for w := 0; w < workers; w++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < opsPerWkr; i++ {
				switch (worker + i) % 4 {
				case 0:
					_, _ = o.PlayNext()
				case 1:
					_, _ = o.PlayBack()
				case 2:
					// progress >= 0.33
					o.ProcessFeedbak(1, 2, 5.0, 10.0)
				case 3:
					// progress < 0.1
					o.ProcessFeedbak(2, 3, 0.5, 10.0)
				}
			}
		}(w)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("concurrent stress did not complete within timeout (elapsed=%s)", time.Since(start))
	}
}

func TestOrchestrator_BaseGraphPenalty_Wrapper(t *testing.T) {
	o := NewOrchestrator()

	bg := o.BaseGraph()
	bg.Reinforce(1, 2)
	bg.Reinforce(1, 2)

	o.baseGraphPenalty(1, 2)

	edges := bg.GetEdges(1)
	if got := edges[2]; got != 1 {
		t.Fatalf("after baseGraphPenalty, BaseGraph().GetEdges(1)[2] = %v, want 1", got)
	}
}

func TestOrchestrator_PlaybackPaths_PlayNextPlayBackForward(t *testing.T) {
	o := NewOrchestrator()
	t.Cleanup(o.Stop)

	bg := o.BaseGraph()

	// Stage 1: deterministic initial pick from fromID=0.
	bg.Reinforce(0, 1)
	o.RebuildRuntime("stage 1")

	id1, ok := o.PlayNext()
	if !ok || id1 != 1 {
		t.Fatalf("PlayNext() stage1 = (%d,%v), want (1,true)", id1, ok)
	}

	// Stage 2: deterministic transition 1 -> 2.
	bg.Reinforce(1, 2)
	o.RebuildRuntime("stage 2")

	id2, ok := o.PlayNext()
	if !ok || id2 != 2 {
		t.Fatalf("PlayNext() stage2 = (%d,%v), want (2,true)", id2, ok)
	}

	// Now back should succeed.
	backID, ok := o.PlayBack()
	if !ok || backID != 1 {
		t.Fatalf("PlayBack() = (%d,%v), want (1,true)", backID, ok)
	}

	// And PlayNext should go forward first.
	fwdID, ok := o.PlayNext()
	if !ok || fwdID != 2 {
		t.Fatalf("PlayNext() forward = (%d,%v), want (2,true)", fwdID, ok)
	}

	// No forward and no outgoing edges from 2 => should fail.
	if id, ok := o.PlayNext(); ok || id != 0 {
		t.Fatalf("PlayNext() from terminal state = (%d,%v), want (0,false)", id, ok)
	}
}

func TestOrchestrator_NilRuntimeGraphPaths(t *testing.T) {
	o := NewOrchestrator()
	o.runtimeGraph.Store(nil)

	if id, ok := o.PlayNext(); ok || id != 0 {
		t.Fatalf("PlayNext() with nil runtimeGraph = (%d,%v), want (0,false)", id, ok)
	}
	o.ProcessFeedbak(1, 2, 1.0, 10.0) // should be a no-op, not a panic
	o.RebuildRuntime("nil current")   // should return early, not panic
}

func TestOrchestrator_manageRuntimeGraphDiffts_RebuildTrigger(t *testing.T) {
	o := NewOrchestrator()
	o.maxRuntimeGraphDiff = 0 // trigger rebuild on first diffts>0 observation
	o.Start()
	defer o.Stop()

	before := o.RuntimeGraph().GetBuildVersion()
	o.ProcessFeedbak(1, 2, 5.0, 10.0) // diffts increments and signals diffChan

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rg := o.RuntimeGraph()
		if rg != nil && rg.GetBuildVersion() > before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected rebuild after diffts trigger; buildVersion stayed at %d", before)
}

func TestOrchestrator_addChainSignal_ContextDoneBranch(t *testing.T) {
	o := NewOrchestrator()
	o.Stop() // cancels lifecycle context

	done := make(chan struct{})
	go func() {
		defer close(done)
		o.addChainSignal(o.diffChan)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("addChainSignal did not return after context cancellation")
	}
}

func TestOrchestrator_addChainSignal_LifecycleNil(t *testing.T) {
	o := NewOrchestrator()
	o.lifecycle.Store(nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		o.addChainSignal(o.diffChan)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("addChainSignal did not return with nil lifecycle")
	}
}

func TestOrchestrator_manageRuntimeGraphTS_LifecycleNilExits(t *testing.T) {
	o := NewOrchestrator()
	o.lifecycle.Store(nil)

	o.wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		o.manageRuntimeGraphTS()
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("manageRuntimeGraphTS did not exit with nil lifecycle")
	}
}

func TestOrchestrator_manageRuntimeGraphDiffts_RuntimeGraphNilBranch(t *testing.T) {
	o := NewOrchestrator()
	o.Start()
	defer o.Stop()

	o.runtimeGraph.Store(nil)
	// Directly signal diffChan (ProcessFeedbak is a no-op when runtimeGraph is nil).
	o.diffChan <- struct{}{}
	time.Sleep(10 * time.Millisecond)
}

func TestOrchestrator_manageRuntimeGraphDiffts_NoRebuildUnderThreshold(t *testing.T) {
	o := NewOrchestrator()
	o.maxRuntimeGraphDiff = 1e12
	o.Start()
	defer o.Stop()

	before := o.RuntimeGraph().GetBuildVersion()
	o.ProcessFeedbak(1, 2, 5.0, 10.0)
	time.Sleep(25 * time.Millisecond)

	after := o.RuntimeGraph().GetBuildVersion()
	if after != before {
		t.Fatalf("unexpected rebuild under high threshold: buildVersion %d -> %d", before, after)
	}
}

func TestOrchestrator_manageRuntimeGraphTS_TTL_Rebuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 60s ticker-driven TTL test in short mode")
	}

	o := NewOrchestrator()
	o.maxRuntimeGraphAge = 0 // TTL immediately exceeded once the ticker fires
	o.Start()
	defer o.Stop()

	deadline := time.Now().Add(65 * time.Second)
	for time.Now().Before(deadline) {
		rg := o.RuntimeGraph()
		if rg != nil && rg.GetBuildVersion() >= 1 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("TTL-driven rebuild did not occur within timeout")
}
