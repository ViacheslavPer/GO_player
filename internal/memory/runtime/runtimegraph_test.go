package runtime

import (
	"math"
	"testing"

	"GO_player/internal/memory/basegraph"
)

const floatTol = 1e-9

func floatEq(a, b float64) bool {
	return math.Abs(a-b) <= floatTol
}

func sumProbs(m map[int64]float64) float64 {
	var s float64
	for _, p := range m {
		s += p
	}
	return s
}

func TestCopyBase_CopiesWeightsCorrectly(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	base.Reinforce(4, 5)
	r := NewRuntimeGraph()
	r.CopyBase(base, 1, "test")
	edges1 := r.GetEdges(1)
	if edges1 == nil || len(edges1) != 2 {
		t.Fatalf("GetEdges(1) = %v, want two targets", edges1)
	}
	if !floatEq(sumProbs(edges1), 1.0) {
		t.Errorf("sum(GetEdges(1)) = %g, want 1.0", sumProbs(edges1))
	}
	if !floatEq(edges1[2], 2.0/3.0) || !floatEq(edges1[3], 1.0/3.0) {
		t.Errorf("GetEdges(1) = %v, want 2:2/3, 3:1/3", edges1)
	}
	edges4 := r.GetEdges(4)
	if edges4 == nil || len(edges4) != 1 || !floatEq(edges4[5], 1.0) {
		t.Errorf("GetEdges(4) = %v, want map[5:1.0]", edges4)
	}
	if v := r.GetBuildVersion(); v != 1 {
		t.Errorf("GetBuildVersion() = %d, want 1", v)
	}
	if r := r.GetBuildReason(); r != "test" {
		t.Errorf("GetBuildReason() = %q, want %q", r, "test")
	}
}

func TestCopyBase_DoesNotShareMaps(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 2 {
		t.Fatalf("after CopyBase: GetEdges(1) = %v", edges)
	}
	base.Reinforce(1, 99)
	edgesAfter := r.GetEdges(1)
	if len(edgesAfter) != 2 {
		t.Errorf("after mutating base: GetEdges(1) has len %d, want 2 (runtime must not share maps)", len(edgesAfter))
	}
	if _, ok := edgesAfter[99]; ok {
		t.Error("GetEdges(1) must not include 99 after base change without second CopyBase")
	}
}

func TestCalculateFines_NoCooldownNoPenalty_EqualsEdges(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	fined := r.calculateFines(1)
	if fined == nil {
		t.Fatal("calculateFines(1) returned nil")
	}
	if fined[2] != 2 || fined[3] != 1 {
		t.Errorf("calculateFines(1) = %v, want map[2:2 3:1]", fined)
	}
}

func TestCalculateFines_AppliesCooldown(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	r.AddCooldown(1, 2, 2)
	fined := r.calculateFines(1)
	if fined == nil {
		t.Fatal("calculateFines(1) returned nil")
	}
	if fined[2] != 1 {
		t.Errorf("after cooldown(1,2,2): fined[2] = %d, want 1 (3-2)", fined[2])
	}
	if fined[3] != 1 {
		t.Errorf("fined[3] = %d, want 1 (no cooldown)", fined[3])
	}
}

func TestCalculateFines_AppliesPenalty(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	r.AddPenalty(1, 3, 1)
	fined := r.calculateFines(1)
	if fined == nil {
		t.Fatal("calculateFines(1) returned nil")
	}
	if fined[2] != 1 {
		t.Errorf("fined[2] = %d, want 1", fined[2])
	}
	if fined[3] != 1 {
		t.Errorf("after penalty(1,3,1): fined[3] = %d, want 1 (2-1)", fined[3])
	}
}

func TestCalculateFines_DoesNotGoNegative(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	r.AddCooldown(1, 2, 10)
	r.AddPenalty(1, 3, 10)
	fined := r.calculateFines(1)
	if fined == nil {
		t.Fatal("calculateFines(1) returned nil")
	}
	if fined[2] < 0 {
		t.Errorf("fined[2] = %d, must not be negative (cooldown > weight)", fined[2])
	}
	if fined[3] < 0 {
		t.Errorf("fined[3] = %d, must not be negative (penalty > weight)", fined[3])
	}
	if fined[2] != 1 {
		t.Errorf("fined[2] = %d, want 1 (1-10 would be negative, so no subtract)", fined[2])
	}
	if fined[3] != 1 {
		t.Errorf("fined[3] = %d, want 1 (1-10 would be negative, so no subtract)", fined[3])
	}
}

func TestCalculateProb_EmptyInput(t *testing.T) {
	prob := calculateProb(nil)
	if prob == nil || len(prob) != 0 {
		t.Errorf("calculateProb(nil) = %v, want non-nil empty map", prob)
	}
	prob = calculateProb(map[int64]int64{})
	if prob == nil || len(prob) != 0 {
		t.Errorf("calculateProb(empty) = %v, want non-nil empty map", prob)
	}
}

func TestCalculateProb_NormalizesCorrectly(t *testing.T) {
	prob := calculateProb(map[int64]int64{2: 1, 4: 1})
	if prob == nil || len(prob) != 2 {
		t.Fatalf("calculateProb = %v", prob)
	}
	if !floatEq(prob[2], 0.5) || !floatEq(prob[4], 0.5) {
		t.Errorf("calculateProb(1,1) = %v, want 0.5, 0.5", prob)
	}
	if !floatEq(sumProbs(prob), 1.0) {
		t.Errorf("sum(prob) = %g, want 1.0", sumProbs(prob))
	}
	prob2 := calculateProb(map[int64]int64{1: 2, 2: 1})
	if !floatEq(prob2[1], 2.0/3.0) || !floatEq(prob2[2], 1.0/3.0) {
		t.Errorf("calculateProb(2,1) = %v, want 2/3, 1/3", prob2)
	}
	if !floatEq(sumProbs(prob2), 1.0) {
		t.Errorf("sum(prob2) = %g, want 1.0", sumProbs(prob2))
	}
}

func TestGetEdges_WithCooldownAndPenalty(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	base.Reinforce(1, 3)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	r.AddCooldown(1, 2, 2)
	r.AddPenalty(1, 3, 1)
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 2 {
		t.Fatalf("GetEdges(1) = %v", edges)
	}
	if !floatEq(sumProbs(edges), 1.0) {
		t.Errorf("sum(GetEdges(1)) = %g, want 1.0", sumProbs(edges))
	}
	want2 := float64(4-2) / float64((4-2)+(3-1))
	want3 := float64(3-1) / float64((4-2)+(3-1))
	if !floatEq(edges[2], want2) || !floatEq(edges[3], want3) {
		t.Errorf("GetEdges(1) = %v, want 2:%g 3:%g (fined then normalized)", edges, want2, want3)
	}
}

func TestGetEdges_EmptyWhenNoEdges(t *testing.T) {
	r := NewRuntimeGraph()
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 0 {
		t.Errorf("GetEdges(1) on empty graph = %v, want non-nil empty map", edges)
	}
}

func TestReduceCooldown_DecrementsAndStopsAtZero(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.CopyBase(base, 0, "")
	r.AddCooldown(1, 2, 2)
	fined0 := r.calculateFines(1)
	if fined0[2] != 1 {
		t.Fatalf("initial fined[2] = %d, want 1 (edge 3 - cooldown 2)", fined0[2])
	}
	r.ReduceCooldown()
	fined1 := r.calculateFines(1)
	if fined1[2] != 2 {
		t.Errorf("after one ReduceCooldown: fined[2] = %d, want 2 (cooldown decremented to 1)", fined1[2])
	}
	r.ReduceCooldown()
	fined2 := r.calculateFines(1)
	if fined2[2] != 3 {
		t.Errorf("after two ReduceCooldown: fined[2] = %d, want 3 (cooldown 0)", fined2[2])
	}
	r.ReduceCooldown()
	r.ReduceCooldown()
	fined4 := r.calculateFines(1)
	if fined4[2] != 3 {
		t.Errorf("after four ReduceCooldown: fined[2] = %d, want 3 (must not go below zero)", fined4[2])
	}
}
