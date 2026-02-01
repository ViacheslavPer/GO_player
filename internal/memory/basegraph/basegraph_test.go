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
