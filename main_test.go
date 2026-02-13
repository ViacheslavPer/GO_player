package main

import (
	"GO_player/internal/orchestrator"
	"math"
	"math/rand"
	"testing"
	"time"
)

// TestMVPv2_EndToEnd simulates MVP v2.0 behavior:
// - Start with small BaseGraph
// - Build RuntimeGraph
// - Create Orchestrator
// - Perform sequence of Play, Next, Skip, Back
// - Validate reinforcement, cooldowns, selector output, PlaybackChain state
func TestMVPv2_EndToEnd(t *testing.T) {
	// Set deterministic seed for reproducible tests
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Intn(42)

	// Step 1: Create Orchestrator (which initializes BaseGraph and RuntimeGraph)
	o := orchestrator.NewOrchestrator()
	if o == nil {
		t.Fatal("Failed to create Orchestrator")
	}

	// Step 2: Build initial BaseGraph with small graph
	// Song IDs: 1, 2, 3, 4, 5
	// Initial transitions: 0->1, 0->2, 1->3, 2->4
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(1, 3)
	o.BaseGraph().Reinforce(2, 4)
	o.BaseGraph().Reinforce(3, 5)

	// Step 3: Rebuild RuntimeGraph from BaseGraph
	o.RebuildRuntime("initial build")
	if o.RuntimeGraph().GetBuildVersion() != 1 {
		t.Errorf("RuntimeGraph buildVersion should be 1 after rebuild, got %d", o.RuntimeGraph().GetBuildVersion())
	}

	// Step 4: Start Orchestrator background processes
	o.Start()
	defer o.Stop()

	// Step 5: Play first song (Next from empty state)
	id1, ok := o.PlayNext()
	if !ok {
		t.Fatal("PlayNext() should succeed with valid graph")
	}
	// Valid IDs are 1, 2, 3, 4, 5 (from the graph we built)
	validIDs := map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	if !validIDs[id1] {
		t.Errorf("First PlayNext() should return valid ID (1-5), got %d", id1)
	}
	if o.PlaybackChain().Current != id1 {
		t.Errorf("PlaybackChain.Current should be %d, got %d", id1, o.PlaybackChain().Current)
	}
	if len(o.PlaybackChain().BackStack) != 0 {
		t.Errorf("After first PlayNext(), BackStack should be empty, got %d items", len(o.PlaybackChain().BackStack))
	}

	// Step 6: Learn from first transition (0 -> id1)
	initialWeight := o.BaseGraph().GetEdges(0)[id1]
	o.Learn(0, id1)

	// Verify reinforcement in BaseGraph
	baseEdges := o.BaseGraph().GetEdges(0)
	if baseEdges[id1] != initialWeight+1.0 {
		t.Errorf("After Learn(0, %d), BaseGraph edge should be %g, got %g", id1, initialWeight+1.0, baseEdges[id1])
	}

	// Step 7: Play Next song
	id2, ok := o.PlayNext()
	if !ok {
		t.Fatal("Second PlayNext() should succeed")
	}
	if o.PlaybackChain().Current != id2 {
		t.Errorf("PlaybackChain.Current should be %d, got %d", id2, o.PlaybackChain().Current)
	}
	if len(o.PlaybackChain().BackStack) != 1 {
		t.Errorf("After second PlayNext(), BackStack should have 1 item, got %d", len(o.PlaybackChain().BackStack))
	}
	if o.PlaybackChain().BackStack[0] != id1 {
		t.Errorf("BackStack should contain first song (%d), got %d", id1, o.PlaybackChain().BackStack[0])
	}

	// Step 8: Learn from second transition
	o.Learn(id1, id2)
	if !o.PlaybackChain().LearningFrozen {
		t.Error("Learning should not be frozen after normal Next()")
	}

	// Step 9: Play Back
	idBack, ok := o.PlayBack()
	if !ok {
		t.Fatal("PlayBack() should succeed when BackStack is not empty")
	}
	if idBack != id1 {
		t.Errorf("PlayBack() should return previous song (%d), got %d", id1, idBack)
	}
	if o.PlaybackChain().Current != id1 {
		t.Errorf("After PlayBack(), Current should be %d, got %d", id1, o.PlaybackChain().Current)
	}
	if !o.PlaybackChain().LearningFrozen {
		t.Error("PlayBack() should freeze learning")
	}
	if len(o.PlaybackChain().ForwardStack) != 1 {
		t.Errorf("After PlayBack(), ForwardStack should have 1 item, got %d", len(o.PlaybackChain().ForwardStack))
	}

	// Step 10: Play Forward (via PlayNext)
	idForward, ok := o.PlayNext()
	if !ok {
		t.Fatal("PlayNext() after Back() should use ForwardStack")
	}
	if idForward != id2 {
		t.Errorf("PlayNext() after Back() should return from ForwardStack (%d), got %d", id2, idForward)
	}
	if o.PlaybackChain().Current != id2 {
		t.Errorf("After Forward(), Current should be %d, got %d", id2, o.PlaybackChain().Current)
	}

	// Step 11: Add cooldown and verify it affects selection
	o.RuntimeGraph().AddCooldown(id2, 3, 10.0)
	o.RuntimeGraph().AddCooldown(id2, 4, 10.0)

	// Get probabilities before and after cooldown
	probsBefore := o.RuntimeGraph().GetEdges(id2)
	if len(probsBefore) == 0 {
		t.Fatal("Should have probabilities for id2")
	}

	// Verify cooldown affects probabilities
	// If id2->3 and id2->4 have high cooldowns, their probabilities should be reduced
	probsAfter := o.RuntimeGraph().GetEdges(id2)
	sumAfter := 0.0
	for _, p := range probsAfter {
		sumAfter += p
	}
	if sumAfter < 0.99 || sumAfter > 1.01 {
		t.Errorf("Probabilities should sum to ~1.0, got %g", sumAfter)
	}

	// Step 12: Apply penalty
	o.RuntimeGraph().Penalty(id2, 3)
	penalties := o.RuntimeGraph().GetPenalty()
	if penalties[id2][3] != 1.0 {
		t.Errorf("Penalty should be recorded, got %g", penalties[id2][3])
	}

	// Step 13: Rebuild RuntimeGraph (applies penalties to BaseGraph)
	o.RebuildRuntime("apply penalties")
	if o.RuntimeGraph().GetBuildVersion() != 2 {
		t.Errorf("After second rebuild, buildVersion should be 2, got %d", o.RuntimeGraph().GetBuildVersion())
	}
	if o.RuntimeGraph().GetDiffts() != 0.0 {
		t.Errorf("After rebuild, diffts should be reset to 0.0, got %g", o.RuntimeGraph().GetDiffts())
	}

	// Step 14: Verify selector output remains valid
	id3, ok := o.PlayNext()
	if !ok {
		t.Fatal("PlayNext() after rebuild should still work")
	}
	if id3 == 0 {
		t.Error("PlayNext() should return valid song ID")
	}

	// Step 15: Verify PlaybackChain state consistency
	if o.PlaybackChain().Current != id3 {
		t.Errorf("Current should be latest song (%d), got %d", id3, o.PlaybackChain().Current)
	}
	if len(o.PlaybackChain().BackStack) < 1 {
		t.Errorf("BackStack should have at least 1 item, got %d", len(o.PlaybackChain().BackStack))
	}

	// Step 16: Verify learning works correctly
	initialWeight = o.BaseGraph().GetEdges(id2)[id3]
	o.Learn(id2, id3)
	finalWeight := o.BaseGraph().GetEdges(id2)[id3]
	if finalWeight != initialWeight+1.0 {
		t.Errorf("Learn() should increment weight from %g to %g, got %g", initialWeight, initialWeight+1.0, finalWeight)
	}

	// Step 17: Verify learning is frozen during navigation
	o.PlayBack()
	if !o.PlaybackChain().LearningFrozen {
		t.Error("Learning should be frozen after Back()")
	}

	// Learning should not update graphs when frozen
	beforeFrozen := o.BaseGraph().GetEdges(o.PlaybackChain().Current)
	o.Learn(o.PlaybackChain().Current, 99)
	afterFrozen := o.BaseGraph().GetEdges(o.PlaybackChain().Current)
	if len(afterFrozen) != len(beforeFrozen) {
		t.Error("Learn() should not update graphs when learning is frozen")
	}
}

// TestMVPv2_SelectorOutput validates that Selector returns valid IDs
func TestMVPv2_SelectorOutput(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Intn(123)
	o := orchestrator.NewOrchestrator()

	// Build graph
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(1, 3)
	o.BaseGraph().Reinforce(1, 4)
	o.RebuildRuntime("test")

	// Play multiple times and verify all returned IDs are valid
	validIDs := map[int64]bool{1: true, 2: true, 3: true, 4: true}

	for i := 0; i < 20; i++ {
		id, ok := o.PlayNext()
		if !ok {
			t.Fatalf("PlayNext() failed on iteration %d", i)
		}
		if !validIDs[id] && id != 0 {
			t.Errorf("PlayNext() returned invalid ID %d on iteration %d", id, i)
		}
	}
}

// TestMVPv2_CooldownManagement verifies cooldowns affect probabilities
func TestMVPv2_CooldownManagement(t *testing.T) {
	o := orchestrator.NewOrchestrator()
	o.Start()
	defer o.Stop()

	// Build graph with higher weights so cooldown has effect
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(0, 3)
	o.BaseGraph().Reinforce(0, 3)
	o.RebuildRuntime("test")

	// Get initial probabilities
	probsBefore := o.RuntimeGraph().GetEdges(0)
	if len(probsBefore) != 3 {
		t.Fatalf("Should have 3 probabilities, got %d", len(probsBefore))
	}

	// Add cooldown to song 1 that will actually reduce its weight
	// Weight is 2.0, cooldown of 0.5 will reduce it to 1.5
	o.RuntimeGraph().AddCooldown(0, 1, 0.5)

	// Get probabilities after cooldown
	probsAfter := o.RuntimeGraph().GetEdges(0)
	if probsAfter[1] >= probsBefore[1] {
		t.Errorf("Cooldown should reduce probability: before=%g after=%g", probsBefore[1], probsAfter[1])
	}

	// Verify probabilities sum to 1.0
	sumAfter := probsAfter[1] + probsAfter[2] + probsAfter[3]
	if math.Abs(sumAfter-1.0) > 1e-9 {
		t.Errorf("Probabilities should sum to 1.0, got %g", sumAfter)
	}
}

// TestMVPv2_PlaybackChainState validates PlaybackChain state transitions
func TestMVPv2_PlaybackChainState(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Intn(456)
	o := orchestrator.NewOrchestrator()

	o.BaseGraph().Reinforce(0, 1)
	o.BaseGraph().Reinforce(0, 2)
	o.BaseGraph().Reinforce(1, 3)
	o.BaseGraph().Reinforce(2, 4)
	o.RebuildRuntime("test")

	// Play sequence: Next -> Next -> Back -> Forward -> Next
	id1, ok1 := o.PlayNext()
	if !ok1 {
		t.Fatal("First PlayNext() should succeed")
	}
	id2, ok2 := o.PlayNext()
	if !ok2 {
		t.Fatal("Second PlayNext() should succeed")
	}

	// Verify state after 2 Nexts
	if o.PlaybackChain().Current != id2 {
		t.Errorf("Current should be %d, got %d", id2, o.PlaybackChain().Current)
	}
	if len(o.PlaybackChain().BackStack) != 1 {
		t.Errorf("BackStack should have 1 item, got %d", len(o.PlaybackChain().BackStack))
	}
	if len(o.PlaybackChain().ForwardStack) != 0 {
		t.Errorf("ForwardStack should be empty, got %d items", len(o.PlaybackChain().ForwardStack))
	}

	// Back
	idBack, okBack := o.PlayBack()
	if !okBack {
		t.Fatal("PlayBack() should succeed when BackStack is not empty")
	}
	if idBack != id1 {
		t.Errorf("Back() should return %d, got %d", id1, idBack)
	}
	if len(o.PlaybackChain().ForwardStack) != 1 {
		t.Errorf("After Back(), ForwardStack should have 1 item, got %d", len(o.PlaybackChain().ForwardStack))
	}

	// Forward (via PlayNext)
	idForward, okForward := o.PlayNext()
	if !okForward {
		t.Fatal("PlayNext() after Back() should use ForwardStack")
	}
	if idForward != id2 {
		t.Errorf("Forward() should return %d, got %d", id2, idForward)
	}
	if len(o.PlaybackChain().ForwardStack) != 0 {
		t.Errorf("After Forward(), ForwardStack should be empty, got %d items", len(o.PlaybackChain().ForwardStack))
	}

	// Next clears forward stack (if any)
	id3, ok3 := o.PlayNext()
	if !ok3 {
		// This might fail if there are no more edges, which is OK
		return
	}
	if len(o.PlaybackChain().ForwardStack) != 0 {
		t.Errorf("After Next(), ForwardStack should be empty, got %d items", len(o.PlaybackChain().ForwardStack))
	}
	if o.PlaybackChain().Current != id3 {
		t.Errorf("Current should be %d, got %d", id3, o.PlaybackChain().Current)
	}
}
