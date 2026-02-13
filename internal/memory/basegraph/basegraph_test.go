package basegraph

import "testing"

func TestNewBaseGraph_InitializesEmptyGraph(t *testing.T) {
	bg := NewBaseGraph()
	if bg == nil {
		t.Fatal("NewBaseGraph() returned nil")
	}
	edges := bg.GetEdges(1)
	if len(edges) != 0 {
		t.Errorf("NewBaseGraph() should return empty graph, got %d edges", len(edges))
	}
	ids := bg.GetAllIDs()
	if len(ids) != 0 {
		t.Errorf("NewBaseGraph() should return empty IDs, got %d IDs", len(ids))
	}
}

func TestReinforce_CreatesEdges(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)

	edges := bg.GetEdges(1)
	if len(edges) != 1 {
		t.Errorf("GetEdges(1) should have 1 edge, got %d", len(edges))
	}
	if edges[10] != 1.0 {
		t.Errorf("Edge 1->10 should have weight 1.0, got %g", edges[10])
	}
}

func TestReinforce_CreatesGlobalEdge(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)

	globalEdges := bg.GetEdges(0)
	if len(globalEdges) != 1 {
		t.Errorf("GetEdges(0) should have 1 edge, got %d", len(globalEdges))
	}
	if globalEdges[10] != 1.0 {
		t.Errorf("Global edge 0->10 should have weight 1.0, got %g", globalEdges[10])
	}
}

func TestReinforce_IncrementsExistingEdge(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Reinforce(1, 10)

	edges := bg.GetEdges(1)
	if edges[10] != 2.0 {
		t.Errorf("Edge 1->10 should have weight 2.0 after 2 reinforces, got %g", edges[10])
	}

	globalEdges := bg.GetEdges(0)
	if globalEdges[10] != 2.0 {
		t.Errorf("Global edge 0->10 should have weight 2.0 after 2 reinforces, got %g", globalEdges[10])
	}
}

func TestReinforce_MultipleTargets(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Reinforce(1, 20)
	bg.Reinforce(1, 10)

	edges := bg.GetEdges(1)
	if len(edges) != 2 {
		t.Errorf("GetEdges(1) should have 2 edges, got %d", len(edges))
	}
	if edges[10] != 2.0 {
		t.Errorf("Edge 1->10 should have weight 2.0, got %g", edges[10])
	}
	if edges[20] != 1.0 {
		t.Errorf("Edge 1->20 should have weight 1.0, got %g", edges[20])
	}
}

func TestPenalty_DecrementsExistingEdge(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Reinforce(1, 10)
	bg.Penalty(1, 10)

	edges := bg.GetEdges(1)
	if edges[10] != 1.0 {
		t.Errorf("Edge 1->10 should have weight 1.0 after penalty, got %g", edges[10])
	}
}

func TestPenalty_DoesNotGoBelowZero(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Penalty(1, 10)
	bg.Penalty(1, 10)

	edges := bg.GetEdges(1)
	if edges[10] != 0.0 {
		t.Errorf("Edge 1->10 should be 0.0 after penalties exceed weight, got %g", edges[10])
	}
}

func TestPenalty_NonExistentEdge_NoOp(t *testing.T) {
	bg := NewBaseGraph()
	bg.Penalty(1, 10)

	edges := bg.GetEdges(1)
	if len(edges) != 0 {
		t.Errorf("Penalty on non-existent edge should not create edge, got %d edges", len(edges))
	}
}

func TestPenalty_NonExistentFromID_NoOp(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Penalty(99, 10)

	edges := bg.GetEdges(99)
	if len(edges) != 0 {
		t.Errorf("Penalty on non-existent fromID should not create edge, got %d edges", len(edges))
	}

	edges1 := bg.GetEdges(1)
	if edges1[10] != 1.0 {
		t.Errorf("Penalty on non-existent fromID should not affect other edges, got %g", edges1[10])
	}
}

func TestGetEdges_NonExistentID_ReturnsEmptyMap(t *testing.T) {
	bg := NewBaseGraph()
	edges := bg.GetEdges(999)
	if len(edges) != 0 {
		t.Errorf("GetEdges(999) should return empty map, got %d edges", len(edges))
	}
}

func TestGetAllIDs_ReturnsAllFromIDs(t *testing.T) {
	bg := NewBaseGraph()
	bg.Reinforce(1, 10)
	bg.Reinforce(2, 20)
	bg.Reinforce(1, 20)

	ids := bg.GetAllIDs()
	if len(ids) != 3 {
		t.Errorf("GetAllIDs() should return 3 IDs (0, 1, 2), got %d", len(ids))
	}

	idMap := make(map[int64]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	if !idMap[0] || !idMap[1] || !idMap[2] {
		t.Errorf("GetAllIDs() should contain 0, 1, 2, got %v", ids)
	}
}

func TestGetAllIDs_EmptyGraph_ReturnsEmptySlice(t *testing.T) {
	bg := NewBaseGraph()
	ids := bg.GetAllIDs()
	if len(ids) != 0 {
		t.Errorf("GetAllIDs() on empty graph should return empty slice, got %d IDs", len(ids))
	}
}
