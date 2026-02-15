package playback

import "testing"

func TestBack_EmptyBackStack_CurrentUnchanged(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 5
	id, ok := pc.Back()
	if ok {
		t.Error("Back() with empty BackStack should return false")
	}
	if id != 0 {
		t.Errorf("Back() should return id=0, got %d", id)
	}
	if pc.Current != 5 {
		t.Errorf("Current should remain 5, got %d", pc.Current)
	}
}

func TestForward_EmptyForwardStack_CurrentUnchanged(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 3
	id, ok := pc.Forward()
	if ok {
		t.Error("Forward() with empty ForwardStack should return false")
	}
	if id != 0 {
		t.Errorf("Forward() should return id=0, got %d", id)
	}
	if pc.Current != 3 {
		t.Errorf("Current should remain 3, got %d", pc.Current)
	}
}

func TestNext_ZeroID_SetsCurrentToZero(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 1
	pc.Next(0)
	if pc.Current != 0 {
		t.Errorf("Next(0) should set Current to 0, got %d", pc.Current)
	}
}

func TestBack_SingleItemBackStack_EmptiesBackStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.BackStack = []int64{1}
	pc.Current = 2
	id, ok := pc.Back()
	if !ok || id != 1 {
		t.Fatalf("Back() want (1, true), got (%d, %v)", id, ok)
	}
	if len(pc.BackStack) != 0 {
		t.Errorf("BackStack should be empty, got len %d", len(pc.BackStack))
	}
	if pc.Current != 1 {
		t.Errorf("Current should be 1, got %d", pc.Current)
	}
}

func TestForward_SingleItemForwardStack_EmptiesForwardStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.ForwardStack = []int64{2}
	pc.Current = 1
	id, ok := pc.Forward()
	if !ok || id != 2 {
		t.Fatalf("Forward() want (2, true), got (%d, %v)", id, ok)
	}
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack should be empty, got len %d", len(pc.ForwardStack))
	}
	if pc.Current != 2 {
		t.Errorf("Current should be 2, got %d", pc.Current)
	}
}

// PlaybackChain is not safe for concurrent use; it is protected by Orchestrator's playbackMutex.
// No concurrent access test here to avoid data race.

func TestFreezeLearning_UnfreezeLearning_Toggle(t *testing.T) {
	pc := &PlaybackChain{}
	pc.FreezeLearning()
	if !pc.LearningFrozen {
		t.Error("FreezeLearning should set LearningFrozen true")
	}
	pc.UnfreezeLearning()
	if pc.LearningFrozen {
		t.Error("UnfreezeLearning should set LearningFrozen false")
	}
	pc.UnfreezeLearning()
	if pc.LearningFrozen {
		t.Error("UnfreezeLearning again should keep false")
	}
}
