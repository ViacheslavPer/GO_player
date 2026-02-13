package runtime

import (
	"GO_player/internal/memory/basegraph"
	"math"
	"testing"
	"time"
)

func TestNewRuntimeGraph_InitializesEmpty(t *testing.T) {
	rg := NewRuntimeGraph()
	if rg == nil {
		t.Fatal("NewRuntimeGraph() returned nil")
	}
	edges := rg.GetEdges(1)
	if len(edges) != 0 {
		t.Errorf("NewRuntimeGraph() should return empty graph, got %d edges", len(edges))
	}
	if rg.GetBuildVersion() != 0 {
		t.Errorf("NewRuntimeGraph() buildVersion should be 0, got %d", rg.GetBuildVersion())
	}
	if rg.GetDiffts() != 0.0 {
		t.Errorf("NewRuntimeGraph() diffts should be 0.0, got %g", rg.GetDiffts())
	}
}

func TestCopyBase_CopiesEdges(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)
	base.Reinforce(1, 20)
	base.Reinforce(2, 30)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 5, "test copy")

	if rg.GetBuildVersion() != 5 {
		t.Errorf("CopyBase() buildVersion should be 5, got %d", rg.GetBuildVersion())
	}
	if rg.GetBuildReason() != "test copy" {
		t.Errorf("CopyBase() buildReason should be 'test copy', got %s", rg.GetBuildReason())
	}

	edges1 := rg.GetEdges(1)
	if len(edges1) != 2 {
		t.Errorf("GetEdges(1) should have 2 edges, got %d", len(edges1))
	}

	edges2 := rg.GetEdges(2)
	if len(edges2) != 1 {
		t.Errorf("GetEdges(2) should have 1 edge, got %d", len(edges2))
	}

	if rg.GetDiffts() != 0.0 {
		t.Errorf("CopyBase() should reset diffts to 0.0, got %g", rg.GetDiffts())
	}
}

func TestCopyBase_SetsTimestamp(t *testing.T) {
	base := basegraph.NewBaseGraph()
	rg := NewRuntimeGraph()
	before := time.Now()
	rg.CopyBase(base, 1, "test")
	after := time.Now()

	timestamp := rg.GetTimestamp()
	if timestamp.Before(before) || timestamp.After(after) {
		t.Errorf("CopyBase() timestamp should be between before and after, got %v", timestamp)
	}
}

func TestReinforce_IncrementsEdges(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.Reinforce(1, 10)
	rg.Reinforce(1, 10)

	edges := rg.GetEdges(1)
	if len(edges) != 1 {
		t.Errorf("GetEdges(1) should have 1 edge, got %d", len(edges))
	}

	globalEdges := rg.GetEdges(0)
	if len(globalEdges) != 1 {
		t.Errorf("GetEdges(0) should have 1 edge, got %d", len(globalEdges))
	}
}

func TestGetEdges_ReturnsProbabilities(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)
	base.Reinforce(1, 20)
	base.Reinforce(1, 20)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 1, "test")

	probs := rg.GetEdges(1)
	if len(probs) != 2 {
		t.Errorf("GetEdges(1) should return 2 probabilities, got %d", len(probs))
	}

	sum := 0.0
	for _, p := range probs {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("Probabilities should sum to 1.0, got %g", sum)
	}

	if probs[20] <= probs[10] {
		t.Errorf("prob[20] should be greater than prob[10] (2 vs 1), got prob[10]=%g prob[20]=%g", probs[10], probs[20])
	}
}

func TestGetEdges_EmptyGraph_ReturnsEmptyMap(t *testing.T) {
	rg := NewRuntimeGraph()
	edges := rg.GetEdges(999)
	if len(edges) != 0 {
		t.Errorf("GetEdges(999) should return empty map, got %d edges", len(edges))
	}
}

func TestAddCooldown_CreatesCooldown(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 1, "test")
	rg.AddCooldown(1, 10, 5.0)

	if rg.GetDiffts() != 1.0 {
		t.Errorf("AddCooldown() should increment diffts to 1.0, got %g", rg.GetDiffts())
	}
}

func TestAddCooldown_AffectsProbabilities(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)
	base.Reinforce(1, 10)
	base.Reinforce(1, 20)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 1, "test")

	probsBefore := rg.GetEdges(1)
	// Add cooldown that is less than the weight (weight is 2.0, cooldown is 0.5)
	rg.AddCooldown(1, 10, 0.5)
	probsAfter := rg.GetEdges(1)

	if probsAfter[10] >= probsBefore[10] {
		t.Errorf("Cooldown should reduce probability of 10, before=%g after=%g", probsBefore[10], probsAfter[10])
	}
	if probsAfter[20] <= probsBefore[20] {
		t.Errorf("Cooldown on 10 should increase probability of 20, before=%g after=%g", probsBefore[20], probsAfter[20])
	}
}

func TestReduceCooldown_DecrementsCooldowns(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.AddCooldown(1, 10, 5.0)
	rg.AddCooldown(1, 20, 3.0)

	initialDiffts := rg.GetDiffts()
	rg.ReduceCooldown()

	if rg.GetDiffts() != initialDiffts-2.0 {
		t.Errorf("ReduceCooldown() should decrement diffts by 2 (one per cooldown), got %g", rg.GetDiffts())
	}
}

func TestReduceCooldown_DoesNotGoBelowZero(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.AddCooldown(1, 10, 1.0)
	rg.ReduceCooldown()
	rg.ReduceCooldown()

	if rg.GetDiffts() < 0 {
		t.Errorf("ReduceCooldown() should not make diffts negative, got %g", rg.GetDiffts())
	}
}

func TestPenalty_IncrementsPenalty(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.Penalty(1, 10)

	penalties := rg.GetPenalty()
	if len(penalties) != 1 {
		t.Errorf("GetPenalty() should have 1 entry, got %d", len(penalties))
	}
	if penalties[1][10] != 1.0 {
		t.Errorf("Penalty(1, 10) should create penalty of 1.0, got %g", penalties[1][10])
	}
	if rg.GetDiffts() != 1.0 {
		t.Errorf("Penalty() should increment diffts to 1.0, got %g", rg.GetDiffts())
	}
}

func TestPenalty_MultiplePenalties_Increment(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.Penalty(1, 10)
	rg.Penalty(1, 10)

	penalties := rg.GetPenalty()
	if penalties[1][10] != 2.0 {
		t.Errorf("Multiple penalties should increment, got %g", penalties[1][10])
	}
	if rg.GetDiffts() != 2.0 {
		t.Errorf("Multiple penalties should increment diffts, got %g", rg.GetDiffts())
	}
}

func TestPenalty_AffectsProbabilities(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)
	base.Reinforce(1, 10)
	base.Reinforce(1, 20)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 1, "test")

	probsBefore := rg.GetEdges(1)
	rg.Penalty(1, 10)
	probsAfter := rg.GetEdges(1)

	if probsAfter[10] >= probsBefore[10] {
		t.Errorf("Penalty should reduce probability of 10, before=%g after=%g", probsBefore[10], probsAfter[10])
	}
}

func TestGetEdges_WithCooldownsAndPenalties(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 10)
	base.Reinforce(1, 10)
	base.Reinforce(1, 20)
	base.Reinforce(1, 20)
	base.Reinforce(1, 30)
	base.Reinforce(1, 30)

	rg := NewRuntimeGraph()
	rg.CopyBase(base, 1, "test")

	// Add cooldown and penalty that will actually affect probabilities
	rg.AddCooldown(1, 10, 0.5) // Reduces weight from 2.0 to 1.5
	rg.Penalty(1, 20)          // Reduces weight from 2.0 to 1.0

	probs := rg.GetEdges(1)
	sum := 0.0
	for _, p := range probs {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("Probabilities should sum to 1.0, got %g", sum)
	}

	// Song 30 should have highest probability (weight 2.0, no cooldown/penalty)
	// Song 10 should have medium probability (weight 1.5 after cooldown)
	// Song 20 should have lowest probability (weight 1.0 after penalty)
	if probs[30] <= probs[10] || probs[30] <= probs[20] {
		t.Errorf("Song 30 should have highest probability, got prob[10]=%g prob[20]=%g prob[30]=%g", probs[10], probs[20], probs[30])
	}
	if probs[20] >= probs[10] {
		t.Errorf("Song 20 (with penalty) should have lower probability than song 10 (with cooldown), got prob[10]=%g prob[20]=%g", probs[10], probs[20])
	}
}

func TestCopyBase_ResetsDiffts(t *testing.T) {
	rg := NewRuntimeGraph()
	rg.AddCooldown(1, 10, 1.0)
	rg.Penalty(1, 20)

	if rg.GetDiffts() != 2.0 {
		t.Fatalf("Setup: diffts should be 2.0, got %g", rg.GetDiffts())
	}

	base := basegraph.NewBaseGraph()
	rg.CopyBase(base, 2, "reset")

	if rg.GetDiffts() != 0.0 {
		t.Errorf("CopyBase() should reset diffts to 0.0, got %g", rg.GetDiffts())
	}
}
