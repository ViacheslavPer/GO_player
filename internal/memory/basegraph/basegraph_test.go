package basegraph

import "testing"

func TestReinforce_CreatesMapsAndIncrementsWeight(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	edges := g.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if w := edges[2]; w != 1 {
		t.Errorf("weight (1,2) = %d, want 1", w)
	}
}

func TestReinforce_IncrementsExistingEdge(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(10, 20)
	g.Reinforce(10, 20)
	edges := g.GetEdges(10)
	if edges == nil {
		t.Fatal("GetEdges(10) returned nil")
	}
	if w := edges[20]; w != 2 {
		t.Errorf("weight (10,20) = %d, want 2", w)
	}
}

func TestReinforce_DifferentFromTo_CreatesSeparateEdges(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Reinforce(1, 3)
	g.Reinforce(4, 5)
	edges1 := g.GetEdges(1)
	if edges1 == nil || edges1[2] != 1 || edges1[3] != 1 {
		t.Errorf("GetEdges(1) = %v, want map[2:1 3:1]", edges1)
	}
	edges4 := g.GetEdges(4)
	if edges4 == nil || edges4[5] != 1 {
		t.Errorf("GetEdges(4) = %v, want map[5:1]", edges4)
	}
}

func TestPenalty_DecrementsWeightByOne(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Reinforce(1, 2)
	g.Penalty(1, 2)
	edges := g.GetEdges(1)
	if w := edges[2]; w != 1 {
		t.Errorf("after Penalty: weight (1,2) = %d, want 1", w)
	}
}

func TestPenalty_NeverGoesBelowZero(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Penalty(1, 2)
	g.Penalty(1, 2)
	g.Penalty(1, 2)
	edges := g.GetEdges(1)
	if w := edges[2]; w != 0 {
		t.Errorf("weight (1,2) = %d, want 0 (floor)", w)
	}
}

func TestPenalty_DoesNothingWhenEdgeDoesNotExist(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Penalty(1, 3)
	edges := g.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if w := edges[2]; w != 1 {
		t.Errorf("weight (1,2) = %d, want 1 (unchanged)", w)
	}
	if edges[3] != 0 {
		t.Errorf("weight (1,3) = %d, want 0", edges[3])
	}
}

func TestPenalty_DoesNothingWhenWeightAlreadyZero(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Penalty(1, 2)
	g.Penalty(1, 2)
	edges := g.GetEdges(1)
	if w := edges[2]; w != 0 {
		t.Errorf("weight (1,2) = %d, want 0", w)
	}
}

func TestGetEdges_ReturnsNonNilEmptyMapWhenNoEdges(t *testing.T) {
	g := NewBaseGraph()
	edges := g.GetEdges(99)
	if edges == nil {
		t.Error("GetEdges(99) returned nil, must not return nil")
	}
	if len(edges) != 0 {
		t.Errorf("GetEdges(99) length = %d, want 0", len(edges))
	}
}

func TestGetEdges_ReturnsCorrectWeights(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 10)
	g.Reinforce(1, 20)
	g.Reinforce(1, 10)
	edges := g.GetEdges(1)
	if edges == nil {
		t.Fatal("GetEdges(1) returned nil")
	}
	if edges[10] != 2 || edges[20] != 1 {
		t.Errorf("GetEdges(1) = %v, want map[10:2 20:1]", edges)
	}
}

func TestGetEdges_UnusedIdReturnsEmptyMap(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	edges := g.GetEdges(42)
	if edges == nil {
		t.Error("GetEdges(42) returned nil, must not return nil")
	}
	if len(edges) != 0 {
		t.Errorf("GetEdges(42) length = %d, want 0", len(edges))
	}
}

func TestGetAllIDs_EmptyGraph_ReturnsNonNilEmptySlice(t *testing.T) {
	g := NewBaseGraph()
	ids := g.GetAllIDs()
	if ids == nil {
		t.Error("GetAllIDs() returned nil, must not return nil")
	}
	if len(ids) != 0 {
		t.Errorf("GetAllIDs() length = %d, want 0", len(ids))
	}
}

func TestGetAllIDs_OneFromID_ReturnsSliceWithThatID(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(42, 1)
	ids := g.GetAllIDs()
	if ids == nil {
		t.Fatal("GetAllIDs() returned nil")
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Errorf("GetAllIDs() = %v, want [42]", ids)
	}
}

func TestGetAllIDs_MultipleFromIDs_ReturnsAllIDs(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 10)
	g.Reinforce(2, 20)
	g.Reinforce(3, 30)
	ids := g.GetAllIDs()
	if ids == nil {
		t.Fatal("GetAllIDs() returned nil")
	}
	if len(ids) != 3 {
		t.Errorf("GetAllIDs() length = %d, want 3", len(ids))
	}
	seen := make(map[int64]bool)
	for _, id := range ids {
		seen[id] = true
	}
	for _, want := range []int64{1, 2, 3} {
		if !seen[want] {
			t.Errorf("GetAllIDs() = %v, missing ID %d", ids, want)
		}
	}
}

func TestGetAllIDs_ReturnedSliceIsIndependent(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Reinforce(3, 4)
	ids := g.GetAllIDs()
	if ids == nil || len(ids) != 2 {
		t.Fatalf("GetAllIDs() = %v", ids)
	}
	ids[0] = 99
	ids[1] = 88
	ids2 := g.GetAllIDs()
	if len(ids2) != 2 {
		t.Errorf("after mutating returned slice: GetAllIDs() length = %d, want 2", len(ids2))
	}
	seen := make(map[int64]bool)
	for _, id := range ids2 {
		seen[id] = true
	}
	if !seen[1] || !seen[3] {
		t.Errorf("after mutating returned slice: GetAllIDs() = %v, want IDs 1 and 3 unchanged in graph", ids2)
	}
}

func TestGetAllIDs_DoesNotMutateGraphState(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Reinforce(3, 4)
	ids1 := g.GetAllIDs()
	ids2 := g.GetAllIDs()
	if len(ids1) != 2 || len(ids2) != 2 {
		t.Fatalf("GetAllIDs() = %v, %v", ids1, ids2)
	}
	edges1 := g.GetEdges(1)
	edges3 := g.GetEdges(3)
	if edges1[2] != 1 || edges3[4] != 1 {
		t.Error("GetAllIDs() must not mutate graph: GetEdges(1)[2] and GetEdges(3)[4] should still be 1")
	}
}
