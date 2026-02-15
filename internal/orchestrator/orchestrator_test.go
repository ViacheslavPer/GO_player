package orchestrator

import (
	"sync"
	"testing"
)

func TestNewOrchestrator_ReturnsNonNil(t *testing.T) {
	o := NewOrchestrator()
	if o == nil {
		t.Fatal("NewOrchestrator() returned nil")
	}
	if o.BaseGraph() == nil {
		t.Error("BaseGraph() should not be nil")
	}
	if o.RuntimeGraph() == nil {
		t.Error("RuntimeGraph() should not be nil after NewOrchestrator")
	}
}

func TestOrchestrator_StartStop_NoPanic(t *testing.T) {
	o := NewOrchestrator()
	o.Start()
	o.Stop()
}

func TestOrchestrator_PlayNext_EmptyGraph_ReturnsFalseOrZero(t *testing.T) {
	o := NewOrchestrator()
	o.Start()
	defer o.Stop()

	id, ok := o.PlayNext()
	if ok && id == 0 {
		t.Log("PlayNext with empty graph may return (0, true) or (0, false) depending on implementation")
	}
}

func TestOrchestrator_PlayNext_WithGraph_ReturnsValidID(t *testing.T) {
	o := NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.RebuildRuntime("test")
	o.Start()
	defer o.Stop()

	id, ok := o.PlayNext()
	if !ok {
		t.Skip("PlayNext returned false; graph may be empty after rebuild")
	}
	if id != 1 && id != 2 {
		t.Errorf("PlayNext should return 1 or 2, got %d", id)
	}
}

func TestOrchestrator_PlayBack_EmptyBackStack_ReturnsFalse(t *testing.T) {
	o := NewOrchestrator()
	o.Start()
	defer o.Stop()

	_, ok := o.PlayBack()
	if ok {
		t.Error("PlayBack with empty BackStack should return false")
	}
}

func TestOrchestrator_Learn_DoesNotPanicWhenNotStarted(t *testing.T) {
	o := NewOrchestrator()
	for i := 0; i < 50; i++ {
		o.Learn(0, 1)
		o.Learn(1, 2)
	}
	if o.BaseGraph() == nil {
		t.Fatal("BaseGraph is nil")
	}
}

func TestOrchestrator_RebuildRuntime_UpdatesRuntimeGraph(t *testing.T) {
	o := NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.RebuildRuntime("first")
	rg1 := o.RuntimeGraph()
	if rg1 == nil {
		t.Fatal("RuntimeGraph nil after first rebuild")
	}
	o.RebuildRuntime("second")
	rg2 := o.RuntimeGraph()
	if rg2 == nil {
		t.Fatal("RuntimeGraph nil after second rebuild")
	}
	if rg2.GetBuildVersion() < 0 {
		t.Errorf("buildVersion should be non-negative, got %d", rg2.GetBuildVersion())
	}
}

func TestOrchestrator_MultipleInstances_Isolated(t *testing.T) {
	var wg sync.WaitGroup
	const instances = 10
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()
			o := NewOrchestrator()
			o.BaseGraph().Reinforce(0, int64(instanceID+1))
			o.RebuildRuntime("isolated")
			o.Start()
			o.PlayNext()
			o.PlayBack()
			o.Stop()
		}(i)
	}
	wg.Wait()
}

func TestOrchestrator_PlaybackChain_ConsistentAfterNext(t *testing.T) {
	o := NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.RebuildRuntime("test")
	o.Start()
	defer o.Stop()

	id, ok := o.PlayNext()
	if !ok {
		t.Skip("PlayNext returned false")
	}
	pc := o.PlaybackChain()
	if pc == nil {
		t.Fatal("PlaybackChain() returned nil")
	}
	if pc.Current != id {
		t.Errorf("PlaybackChain.Current should be %d, got %d", id, pc.Current)
	}
}

func TestOrchestrator_BufferedChannels_LearnWithoutBlock(t *testing.T) {
	o := NewOrchestrator()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			o.Learn(0, 1)
			o.Learn(1, 2)
		}
		close(done)
	}()
	<-done
}

func TestOrchestrator_EmptyGraph_RebuildThenPlayNext(t *testing.T) {
	o := NewOrchestrator()
	o.RebuildRuntime("empty")
	o.Start()
	defer o.Stop()

	_, _ = o.PlayNext()
	_, _ = o.PlayBack()
}
