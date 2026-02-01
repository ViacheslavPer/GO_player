package main

import (
	"GO_player/internal/orchestrator"
	"testing"
)

// Simulator sanity tests: high-level flow only. No mocking, no OS/stdin/flags.
// Asserts non-panicking and ok=true/false after Learn + RebuildRuntime + Next.

func TestSimulatorFlow_LearnRebuildNextReturnsOk(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	o.Learn(1, 2)
	o.RebuildRuntime()
	toID, ok := o.Next(1)
	if !ok {
		t.Errorf("after Learn + RebuildRuntime, Next(1) got ok=false, want ok=true (toID=%d)", toID)
	}
	if toID != 2 {
		t.Errorf("Next(1) toID=%d, want 2", toID)
	}
}

func TestSimulatorFlow_EmptyDataNoPanic(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	toID, ok := o.Next(1)
	if ok {
		t.Errorf("Next(1) on empty graph got ok=true, want ok=false")
	}
	if toID != 0 {
		t.Errorf("Next(1) on empty toID=%d, want 0", toID)
	}
	o.RebuildRuntime()
	toID, ok = o.Next(1)
	if ok {
		t.Errorf("Next(1) after RebuildRuntime on empty got ok=true, want ok=false")
	}
	if toID != 0 {
		t.Errorf("Next(1) after Rebuild on empty toID=%d, want 0", toID)
	}
}

func TestSimulatorFlow_MinimalDataNoPanic(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	o.Learn(1, 2)
	o.RebuildRuntime()
	for i := 0; i < 3; i++ {
		toID, ok := o.Next(1)
		if !ok {
			t.Errorf("Next(1) with minimal data got ok=false on iteration %d", i)
		}
		if toID != 2 {
			t.Errorf("Next(1) toID=%d, want 2 (minimal single edge)", toID)
		}
	}
}
