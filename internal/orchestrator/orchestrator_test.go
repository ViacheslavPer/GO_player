package orchestrator

import (
	"math"
	"testing"
)

func sumProbs(m map[int64]float64) float64 {
	var s float64
	for _, p := range m {
		s += p
	}
	return s
}

func TestNewOrchestrator_InitializesAllComponents(t *testing.T) {
	o := NewOrchestrator()
	if o == nil {
		t.Fatal("NewOrchestrator() returned nil")
	}
	rg := o.GetRuntimeGraph()
	if rg == nil {
		t.Fatal("GetRuntimeGraph() returned nil")
	}
	toID, ok := o.Next(1)
	if ok {
		t.Errorf("Next(1) on empty graph = (%d, true), want (0, false)", toID)
	}
	if toID != 0 {
		t.Errorf("Next(1) toID = %d, want 0", toID)
	}
}

func TestNewOrchestrator_EmptyGraphNextReturnsFalse(t *testing.T) {
	o := NewOrchestrator()
	o.RebuildRuntime()
	toID, ok := o.Next(1)
	if ok {
		t.Errorf("Next(1) after RebuildRuntime on empty = (%d, true), want (0, false)", toID)
	}
	if toID != 0 {
		t.Errorf("Next(1) toID = %d, want 0", toID)
	}
}

func TestLearn_AffectsNextOnlyAfterRebuildRuntime(t *testing.T) {
	o := NewOrchestrator()
	o.Learn(1, 2)
	toID, ok := o.Next(1)
	if ok {
		t.Errorf("Next(1) before RebuildRuntime = (%d, true), want (0, false); Learn must not affect Next until rebuild", toID)
	}
	if toID != 0 {
		t.Errorf("Next(1) toID = %d, want 0", toID)
	}
	o.RebuildRuntime()
	toID, ok = o.Next(1)
	if !ok {
		t.Errorf("Next(1) after RebuildRuntime = (%d, false), want (2, true)", toID)
	}
	if toID != 2 {
		t.Errorf("Next(1) toID = %d, want 2 (single edge)", toID)
	}
}

func TestPenalize_DecreasesInfluenceAfterRebuild(t *testing.T) {
	o := NewOrchestrator()
	o.Learn(1, 2)
	o.Learn(1, 2)
	o.RebuildRuntime()
	toID, ok := o.Next(1)
	if !ok || toID != 2 {
		t.Fatalf("after Learn x2 and Rebuild: Next(1) = (%d, %v), want (2, true)", toID, ok)
	}
	o.Penalize(1, 2)
	o.Penalize(1, 2)
	o.RebuildRuntime()
	toID, ok = o.Next(1)
	if ok {
		t.Errorf("Next(1) after Penalize to zero and Rebuild = (%d, true), want (0, false)", toID)
	}
	if toID != 0 {
		t.Errorf("Next(1) toID = %d, want 0", toID)
	}
}

func TestRebuildRuntime_RebuildsFromCurrentBaseGraph(t *testing.T) {
	o := NewOrchestrator()
	o.Learn(1, 2)
	o.Learn(1, 3)
	o.RebuildRuntime()
	rg := o.GetRuntimeGraph()
	if rg == nil {
		t.Fatal("GetRuntimeGraph() returned nil")
	}
	edges := rg.GetEdges(1)
	if len(edges) != 2 {
		t.Fatalf("GetEdges(1) length = %d, want 2", len(edges))
	}
	if edges[2] == 0 && edges[3] == 0 {
		t.Error("GetEdges(1) should have non-zero probabilities for 2 and 3")
	}
	sum := sumProbs(edges)
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("sum(probs) = %g, want 1.0", sum)
	}
	o.Learn(1, 4)
	o.RebuildRuntime()
	edges = o.GetRuntimeGraph().GetEdges(1)
	if len(edges) != 3 {
		t.Errorf("after Learn(1,4) and Rebuild: GetEdges(1) length = %d, want 3", len(edges))
	}
	if _, ok := edges[4]; !ok {
		t.Error("GetEdges(1) should include 4 after Learn(1,4) and Rebuild")
	}
}

func TestNext_DelegatesToSelector_ReturnsIdOk(t *testing.T) {
	o := NewOrchestrator()
	o.Learn(1, 10)
	o.Learn(1, 20)
	o.RebuildRuntime()
	allowed := map[int64]bool{10: true, 20: true}
	for i := 0; i < 20; i++ {
		toID, ok := o.Next(1)
		if !ok {
			t.Errorf("Next(1) returned false on iteration %d", i)
			continue
		}
		if !allowed[toID] {
			t.Errorf("Next(1) = %d, want 10 or 20", toID)
		}
	}
}

func TestNext_SingleEdge_DeterministicResult(t *testing.T) {
	o := NewOrchestrator()
	o.Learn(1, 42)
	o.RebuildRuntime()
	for i := 0; i < 5; i++ {
		toID, ok := o.Next(1)
		if !ok {
			t.Errorf("Next(1) returned false on iteration %d", i)
		}
		if toID != 42 {
			t.Errorf("Next(1) = %d, want 42 (single edge)", toID)
		}
	}
}

func TestNext_NoPanicOnEmptyGraph(t *testing.T) {
	o := NewOrchestrator()
	toID, ok := o.Next(1)
	if ok || toID != 0 {
		t.Errorf("Next(1) = (%d, %v), want (0, false)", toID, ok)
	}
	o.RebuildRuntime()
	toID, ok = o.Next(1)
	if ok || toID != 0 {
		t.Errorf("Next(1) after Rebuild = (%d, %v), want (0, false)", toID, ok)
	}
}

func TestGetRuntimeGraph_ReturnsCurrentRuntimeGraph(t *testing.T) {
	o := NewOrchestrator()
	rg1 := o.GetRuntimeGraph()
	if rg1 == nil {
		t.Fatal("GetRuntimeGraph() returned nil")
	}
	o.Learn(1, 2)
	o.RebuildRuntime()
	rg2 := o.GetRuntimeGraph()
	if rg2 == nil {
		t.Fatal("GetRuntimeGraph() after Rebuild returned nil")
	}
	if rg1 == rg2 {
		t.Error("GetRuntimeGraph() should return new graph after RebuildRuntime (rebuild replaces graph)")
	}
	edges := rg2.GetEdges(1)
	if len(edges) != 1 || edges[2] != 1.0 {
		t.Errorf("after RebuildRuntime, GetRuntimeGraph().GetEdges(1) = %v, want map[2:1]", edges)
	}
}

func TestOrchestrator_LearnThenRebuild_NoMutationOfRuntimeBeforeRebuild(t *testing.T) {
	o := NewOrchestrator()
	rgBefore := o.GetRuntimeGraph()
	edgesBefore := rgBefore.GetEdges(1)
	if len(edgesBefore) != 0 {
		t.Fatalf("initial GetEdges(1) = %v, want empty", edgesBefore)
	}
	o.Learn(1, 2)
	edgesAfterLearn := o.GetRuntimeGraph().GetEdges(1)
	if len(edgesAfterLearn) != 0 {
		t.Errorf("GetEdges(1) after Learn (no Rebuild) = %v, want empty; runtime must not change until Rebuild", edgesAfterLearn)
	}
}
