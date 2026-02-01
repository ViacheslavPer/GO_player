package selector

import (
	"math"
	"math/rand"
	"testing"

	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
)

func buildRuntimeGraph(edges map[int64]map[int64]int64) *runtime.RuntimeGraph {
	base := basegraph.NewBaseGraph()
	for from, toWeights := range edges {
		for to, weight := range toWeights {
			for i := int64(0); i < weight; i++ {
				base.Reinforce(from, to)
			}
		}
	}
	r := runtime.NewRuntimeGraph()
	r.BuildFromBase(base)
	return r
}

func TestNext_NoEdges_ReturnsFalse(t *testing.T) {
	base := basegraph.NewBaseGraph()
	r := runtime.NewRuntimeGraph()
	r.BuildFromBase(base)
	sel := Newselector(r)
	toID, ok := sel.Next(1)
	if ok {
		t.Errorf("Next(1) = (%d, true), want (0, false)", toID)
	}
	if toID != 0 {
		t.Errorf("Next(1) toID = %d, want 0", toID)
	}
}

func TestNext_FromIDWithNoOutgoingEdges_ReturnsFalse(t *testing.T) {
	r := buildRuntimeGraph(map[int64]map[int64]int64{
		1: {10: 1, 20: 1},
	})
	sel := Newselector(r)
	toID, ok := sel.Next(99)
	if ok {
		t.Errorf("Next(99) = (%d, true), want (0, false)", toID)
	}
	if toID != 0 {
		t.Errorf("Next(99) toID = %d, want 0", toID)
	}
}

func TestComputeGini_UniformDistribution_LowGini(t *testing.T) {
	probs := map[int64]float64{1: 0.5, 2: 0.5}
	gini := computeGini(probs)
	if gini < 0 || gini > 1 {
		t.Errorf("computeGini(probs) = %g, want in [0, 1]", gini)
	}
	expected := 1.0 - (0.25 + 0.25)
	if math.Abs(gini-expected) > 1e-9 {
		t.Errorf("computeGini(uniform 0.5, 0.5) = %g, want %g", gini, expected)
	}
	if gini > 0.5 {
		t.Errorf("uniform distribution should yield gini <= 0.5, got %g", gini)
	}
}

func TestComputeGini_DominantDistribution_HighConcentration(t *testing.T) {
	probs := map[int64]float64{1: 1.0}
	gini := computeGini(probs)
	if gini < 0 || gini > 1 {
		t.Errorf("computeGini(probs) = %g, want in [0, 1]", gini)
	}
	if math.Abs(gini-0) > 1e-9 {
		t.Errorf("computeGini(single 1.0) = %g, want 0", gini)
	}
}

func TestComputeGini_DoesNotMutateInput(t *testing.T) {
	probs := map[int64]float64{1: 0.5, 2: 0.5}
	computeGini(probs)
	if probs[1] != 0.5 || probs[2] != 0.5 {
		t.Error("computeGini must not mutate input map")
	}
}

func TestNext_UsesWeightedWhenGiniBelowThreshold(t *testing.T) {
	rand.New(rand.NewSource(42))
	r := buildRuntimeGraph(map[int64]map[int64]int64{
		1: {10: 1, 20: 1},
	})
	sel := Newselector(r)
	allowed := map[int64]bool{10: true, 20: true}
	for i := 0; i < 50; i++ {
		toID, ok := sel.Next(1)
		if !ok {
			t.Fatalf("Next(1) returned false on iteration %d", i)
		}
		if !allowed[toID] {
			t.Errorf("Next(1) = %d, want 10 or 20", toID)
		}
	}
}

func TestNext_UsesTopKWhenGiniAboveThreshold(t *testing.T) {
	rand.New(rand.NewSource(123))
	r := buildRuntimeGraph(map[int64]map[int64]int64{
		1: {10: 5, 20: 3, 30: 2},
	})
	sel := NewSelectorWithParameters(r, 0.5, 2)
	allowed := map[int64]bool{10: true, 20: true}
	for i := 0; i < 50; i++ {
		toID, ok := sel.Next(1)
		if !ok {
			t.Fatalf("Next(1) returned false on iteration %d", i)
		}
		if !allowed[toID] {
			t.Errorf("Next(1) = %d, want 10 or 20 (top K=2)", toID)
		}
	}
}

func TestSelectTopK_OnlyReturnsFromTopK(t *testing.T) {
	rand.New(rand.NewSource(99))
	probs := map[int64]float64{1: 0.5, 2: 0.3, 3: 0.2}
	allowed := map[int64]bool{1: true, 2: true}
	for i := 0; i < 50; i++ {
		toID, ok := selectTopK(probs, 2)
		if !ok {
			t.Fatalf("selectTopK returned false on iteration %d", i)
		}
		if !allowed[toID] {
			t.Errorf("selectTopK(probs, 2) = %d, want 1 or 2 (top 2)", toID)
		}
	}
}

func TestSelectWeighted_ReturnsOnlyExistingIDs(t *testing.T) {
	rand.New(rand.NewSource(7))
	probs := map[int64]float64{10: 0.5, 20: 0.5}
	allowed := map[int64]bool{10: true, 20: true}
	for i := 0; i < 50; i++ {
		toID, ok := selectWeighted(probs)
		if !ok {
			t.Fatalf("selectWeighted returned false on iteration %d", i)
		}
		if !allowed[toID] {
			t.Errorf("selectWeighted(probs) = %d, want 10 or 20", toID)
		}
	}
}

func TestSelector_DoesNotMutateRuntimeGraph(t *testing.T) {
	r := buildRuntimeGraph(map[int64]map[int64]int64{
		1: {10: 1, 20: 1},
	})
	sel := Newselector(r)
	edgesBefore := r.GetEdges(1)
	sumBefore := 0.0
	for _, p := range edgesBefore {
		sumBefore += p
	}
	rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		sel.Next(1)
	}
	edgesAfter := r.GetEdges(1)
	sumAfter := 0.0
	for _, p := range edgesAfter {
		sumAfter += p
	}
	if math.Abs(sumBefore-1.0) > 1e-9 || math.Abs(sumAfter-1.0) > 1e-9 {
		t.Errorf("RuntimeGraph probabilities changed: sum before %g, after %g", sumBefore, sumAfter)
	}
	if edgesAfter[10] != edgesBefore[10] || edgesAfter[20] != edgesBefore[20] {
		t.Error("RuntimeGraph edge probabilities were mutated by Selector.Next")
	}
}

func TestNewSelectorWithParameters_InvalidValues_AppliesDefaults(t *testing.T) {
	r := runtime.NewRuntimeGraph()
	sel := NewSelectorWithParameters(r, 0.0, 0)
	if sel.giniThreshold != 0.5 {
		t.Errorf("giniThreshold = %g, want 0.5 (default)", sel.giniThreshold)
	}
	if sel.topK != 10 {
		t.Errorf("topK = %d, want 10 (default)", sel.topK)
	}
	sel2 := NewSelectorWithParameters(r, 1.0, -1)
	if sel2.giniThreshold != 0.5 || sel2.topK != 10 {
		t.Errorf("invalid giniThreshold/topK should get defaults, got %g, %d", sel2.giniThreshold, sel2.topK)
	}
}
