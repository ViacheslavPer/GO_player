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

func TestBuildFromBase_EmptyBaseGraph_RuntimeGraphEmpty(t *testing.T) {
	base := basegraph.NewBaseGraph()
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 0 {
		t.Errorf("GetEdges(1) = %v, want non-nil empty map", edges)
	}
}

func TestBuildFromBase_FromIDWithNoEdges_SkipsEntry(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Penalty(1, 2)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 0 {
		t.Errorf("GetEdges(1) = %v, want non-nil empty map (sum=0 skipped)", edges)
	}
}

func TestBuildFromBase_SingleFromIDSingleEdge_ProbabilityOne(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if len(edges) != 1 || !floatEq(edges[2], 1.0) {
		t.Errorf("GetEdges(1) = %v, want map[2:1.0]", edges)
	}
}

func TestBuildFromBase_SingleFromIDMultipleEdges_ProbabilitiesSumToOne(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if len(edges) != 2 {
		t.Errorf("GetEdges(1) has len %d, want 2", len(edges))
	}
	sum := sumProbs(edges)
	if !floatEq(sum, 1.0) {
		t.Errorf("sum(probabilities) = %g, want 1.0", sum)
	}
	if !floatEq(edges[2], 0.5) || !floatEq(edges[3], 0.5) {
		t.Errorf("GetEdges(1) = %v, want 0.5 and 0.5", edges)
	}
}

func TestBuildFromBase_MultipleFromIDs_IndependentNormalization(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	base.Reinforce(4, 5)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges1 := r.GetEdges(1)
	if edges1 == nil || len(edges1) != 2 {
		t.Fatalf("GetEdges(1) = %v", edges1)
	}
	if !floatEq(sumProbs(edges1), 1.0) {
		t.Errorf("fromID 1: sum(probs) = %g, want 1.0", sumProbs(edges1))
	}
	edges4 := r.GetEdges(4)
	if edges4 == nil || len(edges4) != 1 || !floatEq(edges4[5], 1.0) {
		t.Errorf("GetEdges(4) = %v, want map[5:1.0]", edges4)
	}
}

func TestBuildFromBase_ZeroWeightEdge_ZeroProbability(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	base.Reinforce(1, 3)
	base.Penalty(1, 3)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if !floatEq(edges[2], 1.0) || !floatEq(edges[3], 0.0) {
		t.Errorf("GetEdges(1) = %v, want [2:1.0 3:0.0]", edges)
	}
	if !floatEq(sumProbs(edges), 1.0) {
		t.Errorf("sum(probs) = %g, want 1.0", sumProbs(edges))
	}
}

func TestBuildFromBase_RebuildClearsOldProbabilities(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	edges := r.GetEdges(1)
	if edges == nil || !floatEq(edges[2], 1.0) {
		t.Fatalf("first build: GetEdges(1) = %v", edges)
	}

	base.Reinforce(1, 3)
	base.Reinforce(1, 3)
	r.BuildFromBase(base)
	edges = r.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil after rebuild")
	}
	if !floatEq(edges[2], 1.0/3.0) || !floatEq(edges[3], 2.0/3.0) {
		t.Errorf("after rebuild: GetEdges(1) = %v, want [2:1/3 3:2/3]", edges)
	}
	if !floatEq(sumProbs(edges), 1.0) {
		t.Errorf("sum(probs) = %g after rebuild, want 1.0", sumProbs(edges))
	}
}

func TestBuildFromBase_RebuildFromEmptyBase_ClearsRuntime(t *testing.T) {
	base := basegraph.NewBaseGraph()
	base.Reinforce(1, 2)
	r := NewRuntimeGraph()
	r.BuildFromBase(base)
	if r.GetEdges(1) == nil || len(r.GetEdges(1)) == 0 {
		t.Fatal("first build should have edges for 1")
	}

	emptyBase := basegraph.NewBaseGraph()
	r.BuildFromBase(emptyBase)
	edges := r.GetEdges(1)
	if edges == nil || len(edges) != 0 {
		t.Errorf("after rebuild from empty base: GetEdges(1) = %v, want empty", edges)
	}
}
