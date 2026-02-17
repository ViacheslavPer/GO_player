package main

import (
	"GO_player/internal/orchestrator"
	"testing"
)

func TestEndToEnd_PlayNext_Back_Forward(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	t.Cleanup(o.Stop)

	bg := o.BaseGraph()

	// Stage 1: make the initial selection deterministic (fromID=0 has exactly one outgoing option).
	bg.Reinforce(0, 1)
	o.RebuildRuntime("stage 1")

	id1, ok := o.PlayNext()
	if !ok || id1 != 1 {
		t.Fatalf("PlayNext() stage1 = (%d,%v), want (1,true)", id1, ok)
	}

	// Stage 2: add a deterministic transition from 1 -> 2 (initial selection is no longer used).
	bg.Reinforce(1, 2)
	o.RebuildRuntime("stage 2")

	id2, ok := o.PlayNext()
	if !ok || id2 != 2 {
		t.Fatalf("PlayNext() stage2 = (%d,%v), want (2,true)", id2, ok)
	}

	// Feedback should be safe to process end-to-end (no panics / deadlocks).
	o.ProcessFeedbak(1, 2, 5.0, 10.0)

	backID, ok := o.PlayBack()
	if !ok || backID != 1 {
		t.Fatalf("PlayBack() = (%d,%v), want (1,true)", backID, ok)
	}

	// PlayNext should prefer forward navigation if available.
	fwdID, ok := o.PlayNext()
	if !ok || fwdID != 2 {
		t.Fatalf("PlayNext() after back (forward) = (%d,%v), want (2,true)", fwdID, ok)
	}
}

func TestMain_DoesNotPanic(t *testing.T) {
	main()
}
