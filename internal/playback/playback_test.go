package playback

import "testing"

func TestNext_PushesCurrentToBackStackAndReturnsIdOk(t *testing.T) {
	pc := &PlaybackChain{Current: 10}
	id, ok := pc.Next(20)
	if id != 20 || !ok {
		t.Errorf("Next(20) = (%d, %v), want (20, true)", id, ok)
	}
	if pc.Current != 20 {
		t.Errorf("Current = %d, want 20", pc.Current)
	}
	if len(pc.BackStack) != 1 || pc.BackStack[0] != 10 {
		t.Errorf("BackStack = %v, want [10]", pc.BackStack)
	}
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack = %v, want empty", pc.ForwardStack)
	}
}

func TestNext_WhenCurrentZero_DoesNotPushToBackStack(t *testing.T) {
	pc := &PlaybackChain{Current: 0}
	id, ok := pc.Next(5)
	if id != 5 || !ok {
		t.Errorf("Next(5) = (%d, %v), want (5, true)", id, ok)
	}
	if pc.Current != 5 {
		t.Errorf("Current = %d, want 5", pc.Current)
	}
	if len(pc.BackStack) != 0 {
		t.Errorf("BackStack = %v, want empty when Current was 0", pc.BackStack)
	}
}

func TestNext_ClearsForwardStack(t *testing.T) {
	pc := &PlaybackChain{Current: 1, ForwardStack: []int64{2, 3}}
	pc.Next(4)
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack = %v, want empty after Next", pc.ForwardStack)
	}
	if pc.Current != 4 {
		t.Errorf("Current = %d, want 4", pc.Current)
	}
}

func TestBack_Success_ReturnsNewCurrentAndTrue(t *testing.T) {
	pc := &PlaybackChain{BackStack: []int64{100}, Current: 200}
	id, ok := pc.Back()
	if id != 100 || !ok {
		t.Errorf("Back() = (%d, %v), want (100, true)", id, ok)
	}
	if pc.Current != 100 {
		t.Errorf("Current = %d, want 100", pc.Current)
	}
	if len(pc.BackStack) != 0 {
		t.Errorf("BackStack = %v, want empty", pc.BackStack)
	}
	if len(pc.ForwardStack) != 1 || pc.ForwardStack[0] != 200 {
		t.Errorf("ForwardStack = %v, want [200]", pc.ForwardStack)
	}
}

func TestBack_EmptyBackStack_ReturnsZeroFalseAndNoStateChange(t *testing.T) {
	pc := &PlaybackChain{BackStack: nil, Current: 5}
	id, ok := pc.Back()
	if id != 0 || ok {
		t.Errorf("Back() = (%d, %v), want (0, false)", id, ok)
	}
	if pc.Current != 5 {
		t.Errorf("Current = %d, want 5 (unchanged)", pc.Current)
	}
	if len(pc.BackStack) != 0 || len(pc.ForwardStack) != 0 {
		t.Errorf("stacks changed: BackStack=%v ForwardStack=%v", pc.BackStack, pc.ForwardStack)
	}
}

func TestBack_CurrentZero_ReturnsZeroFalseAndNoStateChange(t *testing.T) {
	pc := &PlaybackChain{BackStack: []int64{1, 2}, Current: 0}
	id, ok := pc.Back()
	if id != 0 || ok {
		t.Errorf("Back() = (%d, %v), want (0, false)", id, ok)
	}
	if pc.Current != 0 {
		t.Errorf("Current = %d, want 0 (unchanged)", pc.Current)
	}
	if len(pc.BackStack) != 2 {
		t.Errorf("BackStack = %v, want [1 2] unchanged", pc.BackStack)
	}
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack = %v, want empty", pc.ForwardStack)
	}
}

func TestForward_Success_ReturnsNewCurrentAndTrue(t *testing.T) {
	pc := &PlaybackChain{ForwardStack: []int64{300}, Current: 200}
	id, ok := pc.Forward()
	if id != 300 || !ok {
		t.Errorf("Forward() = (%d, %v), want (300, true)", id, ok)
	}
	if pc.Current != 300 {
		t.Errorf("Current = %d, want 300", pc.Current)
	}
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack = %v, want empty", pc.ForwardStack)
	}
	if len(pc.BackStack) != 1 || pc.BackStack[0] != 200 {
		t.Errorf("BackStack = %v, want [200]", pc.BackStack)
	}
}

func TestForward_EmptyForwardStack_ReturnsZeroFalseAndNoStateChange(t *testing.T) {
	pc := &PlaybackChain{ForwardStack: nil, Current: 5}
	id, ok := pc.Forward()
	if id != 0 || ok {
		t.Errorf("Forward() = (%d, %v), want (0, false)", id, ok)
	}
	if pc.Current != 5 {
		t.Errorf("Current = %d, want 5 (unchanged)", pc.Current)
	}
	if len(pc.ForwardStack) != 0 || len(pc.BackStack) != 0 {
		t.Errorf("stacks changed: BackStack=%v ForwardStack=%v", pc.BackStack, pc.ForwardStack)
	}
}

func TestForward_CurrentZero_PushesNothingToBackStackButStillPops(t *testing.T) {
	pc := &PlaybackChain{ForwardStack: []int64{7}, Current: 0}
	id, ok := pc.Forward()
	if id != 7 || !ok {
		t.Errorf("Forward() = (%d, %v), want (7, true)", id, ok)
	}
	if pc.Current != 7 {
		t.Errorf("Current = %d, want 7", pc.Current)
	}
	if len(pc.BackStack) != 0 {
		t.Errorf("BackStack = %v, want empty (Current was 0)", pc.BackStack)
	}
	if len(pc.ForwardStack) != 0 {
		t.Errorf("ForwardStack = %v, want empty", pc.ForwardStack)
	}
}

func TestBackAndForward_Sequence(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Next(1)
	pc.Next(2)
	pc.Next(3)
	if pc.Current != 3 || len(pc.BackStack) != 2 {
		t.Fatalf("setup: Current=%d BackStack=%v", pc.Current, pc.BackStack)
	}
	id, ok := pc.Back()
	if id != 2 || !ok {
		t.Errorf("Back() = (%d, %v), want (2, true)", id, ok)
	}
	id, ok = pc.Back()
	if id != 1 || !ok {
		t.Errorf("Back() = (%d, %v), want (1, true)", id, ok)
	}
	id, ok = pc.Forward()
	if id != 2 || !ok {
		t.Errorf("Forward() = (%d, %v), want (2, true)", id, ok)
	}
	id, ok = pc.Forward()
	if id != 3 || !ok {
		t.Errorf("Forward() = (%d, %v), want (3, true)", id, ok)
	}
	if pc.Current != 3 || len(pc.BackStack) != 2 || len(pc.ForwardStack) != 0 {
		t.Errorf("final: Current=%d BackStack=%v ForwardStack=%v", pc.Current, pc.BackStack, pc.ForwardStack)
	}
}

func TestFreezeLearning_SetsFlagTrue(t *testing.T) {
	pc := &PlaybackChain{LearningFrozen: false}
	pc.FreezeLearning()
	if !pc.LearningFrozen {
		t.Error("LearningFrozen want true after FreezeLearning()")
	}
}

func TestUnfreezeLearning_SetsFlagFalse(t *testing.T) {
	pc := &PlaybackChain{LearningFrozen: true}
	pc.UnfreezeLearning()
	if pc.LearningFrozen {
		t.Error("LearningFrozen want false after UnfreezeLearning()")
	}
}

func TestFreezeLearningUnfreezeLearning_Toggle(t *testing.T) {
	pc := &PlaybackChain{}
	pc.FreezeLearning()
	if !pc.LearningFrozen {
		t.Error("LearningFrozen want true after FreezeLearning()")
	}
	pc.UnfreezeLearning()
	if pc.LearningFrozen {
		t.Error("LearningFrozen want false after UnfreezeLearning()")
	}
}
