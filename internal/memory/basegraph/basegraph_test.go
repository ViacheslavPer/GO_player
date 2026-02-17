package basegraph

import "testing"

func TestBaseGraph_ReinforcePenalty_GetEdgesCopy(t *testing.T) {
	g := NewBaseGraph()

	// Reinforce creates both per-from edges and global edges (fromID=0).
	g.Reinforce(1, 2)
	g.Reinforce(1, 2)

	edges1 := g.GetEdges(1)
	if got := edges1[2]; got != 2 {
		t.Fatalf("GetEdges(1)[2] = %v, want 2", got)
	}

	edges0 := g.GetEdges(0)
	if got := edges0[2]; got != 2 {
		t.Fatalf("GetEdges(0)[2] = %v, want 2", got)
	}

	// GetEdges must return a copy (mutating the result must not mutate internal state).
	edges1[2] = 999
	edgesAgain := g.GetEdges(1)
	if got := edgesAgain[2]; got != 2 {
		t.Fatalf("GetEdges(1) after external mutation = %v, want 2", got)
	}

	// Penalty decrements but must not go below zero.
	g.Penalty(1, 2)
	g.Penalty(1, 2)
	g.Penalty(1, 2) // extra

	edges1 = g.GetEdges(1)
	if got := edges1[2]; got != 0 {
		t.Fatalf("after penalties, GetEdges(1)[2] = %v, want 0", got)
	}
	edges0 = g.GetEdges(0)
	if got := edges0[2]; got != 0 {
		t.Fatalf("after penalties, GetEdges(0)[2] = %v, want 0", got)
	}

	// Penalty on missing fromID must be a no-op.
	g.Penalty(123, 456)
	if got := len(g.GetEdges(123)); got != 0 {
		t.Fatalf("GetEdges(123) size = %d, want 0", got)
	}
}

func TestBaseGraph_GetAllIDs(t *testing.T) {
	g := NewBaseGraph()
	g.Reinforce(1, 2)
	g.Reinforce(3, 4)

	ids := g.GetAllIDs()
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}

	// Reinforce always touches fromID and fromID=0.
	for _, want := range []int64{0, 1, 3} {
		if !seen[want] {
			t.Fatalf("GetAllIDs missing id %d; got %v", want, ids)
		}
	}
}
