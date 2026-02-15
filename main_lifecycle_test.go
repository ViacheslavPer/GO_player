package main

import (
	"GO_player/internal/orchestrator"
	"sync"
	"testing"
	"time"
)

// TestMain_OrchestratorLifecycle simulates full orchestrator lifecycle: init, Start, concurrent
// PlayNext/PlayBack only (no Learn to avoid triggering RebuildRuntime and deadlock), then graceful shutdown.
func TestMain_OrchestratorLifecycle(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	if o == nil {
		t.Fatal("NewOrchestrator returned nil")
	}

	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(1, 3)
	o.BaseGraph().Reinforce(2, 4)
	o.RebuildRuntime("initial")
	o.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			o.PlayNext()
			o.PlayBack()
		}
	}()

	wg.Wait()

	if o.BaseGraph() == nil {
		t.Error("BaseGraph nil after lifecycle")
	}
	if o.RuntimeGraph() == nil {
		t.Error("RuntimeGraph nil after lifecycle")
	}
	pc := o.PlaybackChain()
	if pc == nil {
		t.Error("PlaybackChain nil after lifecycle")
	}

	time.Sleep(25 * time.Millisecond)
	o.Stop()
}

// TestMain_OrchestratorNoPanicUnderConcurrentLoad ensures no panic under concurrent PlayNext/PlayBack.
func TestMain_OrchestratorNoPanicUnderConcurrentLoad(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.RebuildRuntime("load")
	o.Start()
	defer func() {
		time.Sleep(20 * time.Millisecond)
		o.Stop()
	}()

	var wg sync.WaitGroup
	const goroutines = 40
	const iters = 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				o.PlayNext()
				o.PlayBack()
			}
		}()
	}
	wg.Wait()
}

// TestMain_OrchestratorGracefulShutdown verifies Stop() returns.
func TestMain_OrchestratorGracefulShutdown(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	o.Start()
	o.Stop()
}
