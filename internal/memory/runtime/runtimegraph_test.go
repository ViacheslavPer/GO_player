package runtime

import (
	"GO_player/internal/memory/basegraph"
	"math"
	"testing"
	"time"
)

func TestRuntimeGraph_BuildRebuildCopyBase_Getters(t *testing.T) {
	bg := basegraph.NewBaseGraph()
	bg.Reinforce(1, 2)

	rg := NewRuntimeGraph()
	rg.BuildFromBase(bg)

	if got := rg.GetBuildVersion(); got != 0 {
		t.Fatalf("GetBuildVersion() after BuildFromBase = %d, want 0", got)
	}
	if ts := rg.GetTimestamp(); ts.IsZero() {
		t.Fatalf("GetTimestamp() is zero after BuildFromBase")
	}
	if got := rg.GetDiffts(); got != 0 {
		t.Fatalf("GetDiffts() after BuildFromBase = %v, want 0", got)
	}

	// RebuildFromBase increments buildVersion and resets diffts.
	rg.Reinforce(1, 2, 1)
	if got := rg.GetDiffts(); got == 0 {
		t.Fatalf("GetDiffts() after Reinforce = %v, want > 0", got)
	}
	rg.RebuildFromBase(bg, "test rebuild")
	if got := rg.GetBuildVersion(); got != 1 {
		t.Fatalf("GetBuildVersion() after RebuildFromBase = %d, want 1", got)
	}
	if got := rg.GetBuildReason(); got == "" {
		t.Fatalf("GetBuildReason() is empty after RebuildFromBase")
	}
	if got := rg.GetDiffts(); got != 0 {
		t.Fatalf("GetDiffts() after RebuildFromBase = %v, want 0", got)
	}

	// CopyBase must set the requested version/reason.
	rg.CopyBase(bg, 7, "copy base reason")
	if got := rg.GetBuildVersion(); got != 7 {
		t.Fatalf("GetBuildVersion() after CopyBase = %d, want 7", got)
	}
	if got := rg.GetBuildReason(); got != "copy base reason" {
		t.Fatalf("GetBuildReason() after CopyBase = %q, want %q", got, "copy base reason")
	}
}

func TestRuntimeGraph_GetEdges_ProbabilitiesAndFines(t *testing.T) {
	rg := NewRuntimeGraph()

	// Construct two outgoing edges with different weights.
	rg.Reinforce(1, 2, 10)
	rg.Reinforce(1, 3, 10)

	// Apply a cooldown and a penalty; GetEdges should still be a proper probability distribution.
	rg.AddCooldown(1, 2, 0.2)
	rg.Penalty(1, 3, 1.0)

	probs := rg.GetEdges(1)
	if len(probs) != 2 {
		t.Fatalf("GetEdges(1) size = %d, want 2", len(probs))
	}
	if _, ok := probs[2]; !ok {
		t.Fatalf("GetEdges(1) missing toID=2; got %v", probs)
	}
	if _, ok := probs[3]; !ok {
		t.Fatalf("GetEdges(1) missing toID=3; got %v", probs)
	}

	sum := 0.0
	for _, p := range probs {
		if math.IsNaN(p) || p < 0 {
			t.Fatalf("invalid probability %v in %v", p, probs)
		}
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Fatalf("probability sum = %.12f, want 1.0 (probs=%v)", sum, probs)
	}
}

func TestRuntimeGraph_AddCooldown_InvalidValueCanZeroOutDistribution(t *testing.T) {
	rg := NewRuntimeGraph()

	// Small weight, then an invalid cooldown value which is clamped internally.
	rg.Reinforce(1, 2, 0.5)
	rg.AddCooldown(1, 2, -1.0)

	// With a strong cooldown applied immediately, the fined weight can become 0,
	// which should yield an empty probability map (sum==0).
	probs := rg.GetEdges(1)
	if len(probs) != 0 {
		t.Fatalf("GetEdges(1) = %v, want empty map", probs)
	}
}

func TestRuntimeGraph_GetPenalty_ReturnsDeepCopy(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.Penalty(1, 2, 3.0)

	p1 := rg.GetPenalty()
	if p1[1][2] != 3.0 {
		t.Fatalf("GetPenalty()[1][2] = %v, want 3.0", p1[1][2])
	}

	// Mutate returned map; internal state must not change.
	p1[1][2] = 999
	p2 := rg.GetPenalty()
	if p2[1][2] != 3.0 {
		t.Fatalf("GetPenalty()[1][2] after external mutation = %v, want 3.0", p2[1][2])
	}
}

func TestRuntimeGraph_GetEdges_ConcurrentReadWhileWriting(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.Reinforce(1, 2, 1)
	rg.Reinforce(1, 3, 1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(150 * time.Millisecond)
		for time.Now().Before(deadline) {
			rg.Reinforce(1, 2, 1)
			rg.Penalty(1, 3, 0.1)
			rg.AddCooldown(1, 2, 0.2)
		}
	}()

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		_ = rg.GetEdges(1)
		_ = rg.GetPenalty()
	}
	<-done
}

func Test_calculateProb_NilInput(t *testing.T) {
	got := calculateProb(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("calculateProb(nil) = %v, want empty (non-nil) map", got)
	}
}

func Test_RuntimeGraph_calculateFines_EdgeMissingAndPenaltyNoOverSubtract(t *testing.T) {
	rg := NewRuntimeGraph()

	rg.mu.RLock()
	empty := rg.calculateFines(999)
	rg.mu.RUnlock()
	if len(empty) != 0 {
		t.Fatalf("calculateFines(missing) = %v, want empty", empty)
	}

	// If penalty exceeds weight, calculateFines should not make the weight negative.
	rg.Reinforce(1, 2, 0.5)
	rg.Penalty(1, 2, 10.0)

	probs := rg.GetEdges(1)
	// With one edge and a large penalty, the fined weight remains non-negative.
	for _, p := range probs {
		if math.IsNaN(p) || p < 0 {
			t.Fatalf("invalid probability %v in %v", p, probs)
		}
	}
}

func TestRuntimeGraph_GetEdges_MissingFromIDReturnsEmpty(t *testing.T) {
	rg := NewRuntimeGraph()
	got := rg.GetEdges(999)
	if got == nil || len(got) != 0 {
		t.Fatalf("GetEdges(missing) = %v, want empty (non-nil) map", got)
	}
}
