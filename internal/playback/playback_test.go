package playback

import "testing"

func TestNext_PushesCurrentToBackStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 1

	pc.Next(2)

	if pc.Current != 2 {
		t.Errorf("Next(2) should set Current to 2, got %d", pc.Current)
	}
	if len(pc.BackStack) != 1 {
		t.Errorf("Next() should push previous Current to BackStack, got %d items", len(pc.BackStack))
	}
	if pc.BackStack[0] != 1 {
		t.Errorf("BackStack should contain previous Current (1), got %d", pc.BackStack[0])
	}
}

func TestNext_ClearsForwardStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.ForwardStack = []int64{10, 20}
	pc.Current = 1

	pc.Next(2)

	if len(pc.ForwardStack) != 0 {
		t.Errorf("Next() should clear ForwardStack, got %d items", len(pc.ForwardStack))
	}
}

func TestNext_ZeroCurrent_DoesNotPushToBackStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 0

	pc.Next(1)

	if len(pc.BackStack) != 0 {
		t.Errorf("Next() with Current=0 should not push to BackStack, got %d items", len(pc.BackStack))
	}
	if pc.Current != 1 {
		t.Errorf("Next(1) should set Current to 1, got %d", pc.Current)
	}
}

func TestNext_MultipleNexts_BuildsBackStack(t *testing.T) {
	pc := &PlaybackChain{}

	pc.Next(1)
	pc.Next(2)
	pc.Next(3)

	if pc.Current != 3 {
		t.Errorf("Current should be 3, got %d", pc.Current)
	}
	if len(pc.BackStack) != 2 {
		t.Errorf("BackStack should have 2 items, got %d", len(pc.BackStack))
	}
	if pc.BackStack[0] != 1 || pc.BackStack[1] != 2 {
		t.Errorf("BackStack should be [1, 2], got %v", pc.BackStack)
	}
}

func TestBack_ReturnsPreviousSong(t *testing.T) {
	pc := &PlaybackChain{}
	pc.BackStack = []int64{1, 2}
	pc.Current = 3

	id, ok := pc.Back()

	if !ok {
		t.Fatal("Back() should return true when BackStack is not empty")
	}
	if id != 2 {
		t.Errorf("Back() should return last item from BackStack (2), got %d", id)
	}
	if pc.Current != 2 {
		t.Errorf("Back() should set Current to 2, got %d", pc.Current)
	}
	if len(pc.BackStack) != 1 {
		t.Errorf("Back() should remove last item from BackStack, got %d items", len(pc.BackStack))
	}
}

func TestBack_PushesCurrentToForwardStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.BackStack = []int64{1}
	pc.Current = 2

	pc.Back()

	if len(pc.ForwardStack) != 1 {
		t.Errorf("Back() should push Current to ForwardStack, got %d items", len(pc.ForwardStack))
	}
	if pc.ForwardStack[0] != 2 {
		t.Errorf("ForwardStack should contain previous Current (2), got %d", pc.ForwardStack[0])
	}
}

func TestBack_EmptyBackStack_ReturnsFalse(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 1

	id, ok := pc.Back()

	if ok {
		t.Errorf("Back() with empty BackStack should return false, got true")
	}
	if id != 0 {
		t.Errorf("Back() with empty BackStack should return id=0, got %d", id)
	}
	if pc.Current != 1 {
		t.Errorf("Back() with empty BackStack should not change Current, got %d", pc.Current)
	}
	_ = id // Use id to avoid unused variable
}

func TestBack_ZeroCurrent_ReturnsFalse(t *testing.T) {
	pc := &PlaybackChain{}
	pc.BackStack = []int64{1}
	pc.Current = 0

	id, ok := pc.Back()

	if ok {
		t.Errorf("Back() with Current=0 should return false, got true")
	}
	if id != 0 {
		t.Errorf("Back() with Current=0 should return id=0, got %d", id)
	}
}

func TestForward_ReturnsNextSong(t *testing.T) {
	pc := &PlaybackChain{}
	pc.ForwardStack = []int64{2, 3}
	pc.Current = 1

	id, ok := pc.Forward()

	if !ok {
		t.Fatal("Forward() should return true when ForwardStack is not empty")
	}
	if id != 3 {
		t.Errorf("Forward() should return last item from ForwardStack (3), got %d", id)
	}
	if pc.Current != 3 {
		t.Errorf("Forward() should set Current to 3, got %d", pc.Current)
	}
	if len(pc.ForwardStack) != 1 {
		t.Errorf("Forward() should remove last item from ForwardStack, got %d items", len(pc.ForwardStack))
	}
}

func TestForward_PushesCurrentToBackStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.ForwardStack = []int64{2}
	pc.Current = 1

	pc.Forward()

	if len(pc.BackStack) != 1 {
		t.Errorf("Forward() should push Current to BackStack, got %d items", len(pc.BackStack))
	}
	if pc.BackStack[0] != 1 {
		t.Errorf("BackStack should contain previous Current (1), got %d", pc.BackStack[0])
	}
}

func TestForward_ZeroCurrent_DoesNotPushToBackStack(t *testing.T) {
	pc := &PlaybackChain{}
	pc.ForwardStack = []int64{1}
	pc.Current = 0

	id, ok := pc.Forward()

	if !ok {
		t.Fatal("Forward() should return true when ForwardStack is not empty")
	}
	if id != 1 {
		t.Errorf("Forward() should return 1, got %d", id)
	}
	if len(pc.BackStack) != 0 {
		t.Errorf("Forward() with Current=0 should not push to BackStack, got %d items", len(pc.BackStack))
	}
	if pc.Current != 1 {
		t.Errorf("Forward() should set Current to 1, got %d", pc.Current)
	}
}

func TestForward_EmptyForwardStack_ReturnsFalse(t *testing.T) {
	pc := &PlaybackChain{}
	pc.Current = 1

	id, ok := pc.Forward()

	if ok {
		t.Errorf("Forward() with empty ForwardStack should return false, got true")
	}
	if id != 0 {
		t.Errorf("Forward() with empty ForwardStack should return id=0, got %d", id)
	}
	if pc.Current != 1 {
		t.Errorf("Forward() with empty ForwardStack should not change Current, got %d", pc.Current)
	}
}

func TestFreezeLearning_SetsFlag(t *testing.T) {
	pc := &PlaybackChain{}
	pc.LearningFrozen = false

	pc.FreezeLearning()

	if !pc.LearningFrozen {
		t.Error("FreezeLearning() should set LearningFrozen to true")
	}
}

func TestFreezeLearning_AlreadyFrozen_NoOp(t *testing.T) {
	pc := &PlaybackChain{}
	pc.LearningFrozen = true

	pc.FreezeLearning()

	if !pc.LearningFrozen {
		t.Error("FreezeLearning() should keep LearningFrozen as true")
	}
}

func TestUnfreezeLearning_ClearsFlag(t *testing.T) {
	pc := &PlaybackChain{}
	pc.LearningFrozen = true

	pc.UnfreezeLearning()

	if pc.LearningFrozen {
		t.Error("UnfreezeLearning() should set LearningFrozen to false")
	}
}

func TestUnfreezeLearning_AlreadyUnfrozen_NoOp(t *testing.T) {
	pc := &PlaybackChain{}
	pc.LearningFrozen = false

	pc.UnfreezeLearning()

	if pc.LearningFrozen {
		t.Error("UnfreezeLearning() should keep LearningFrozen as false")
	}
}

func TestPlaybackChain_NavigationFlow(t *testing.T) {
	pc := &PlaybackChain{}

	// Start: Current=0
	pc.Next(1) // BackStack=[], Current=1, ForwardStack=[]
	pc.Next(2) // BackStack=[1], Current=2, ForwardStack=[]
	pc.Next(3) // BackStack=[1,2], Current=3, ForwardStack=[]

	if pc.Current != 3 {
		t.Fatalf("After 3 Next() calls, Current should be 3, got %d", pc.Current)
	}
	if len(pc.BackStack) != 2 {
		t.Fatalf("BackStack should have 2 items, got %d", len(pc.BackStack))
	}

	// Go back
	id, ok := pc.Back() // BackStack=[1], Current=2, ForwardStack=[3]
	if !ok || id != 2 {
		t.Fatalf("Back() should return 2, got (%d, %v)", id, ok)
	}

	id, ok = pc.Back() // BackStack=[], Current=1, ForwardStack=[3,2]
	if !ok || id != 1 {
		t.Fatalf("Back() should return 1, got (%d, %v)", id, ok)
	}

	// Go forward
	id, ok = pc.Forward() // BackStack=[1], Current=2, ForwardStack=[3]
	if !ok || id != 2 {
		t.Fatalf("Forward() should return 2, got (%d, %v)", id, ok)
	}

	id, ok = pc.Forward() // BackStack=[1,2], Current=3, ForwardStack=[]
	if !ok || id != 3 {
		t.Fatalf("Forward() should return 3, got (%d, %v)", id, ok)
	}

	// Next clears forward stack
	pc.Next(4) // BackStack=[1,2,3], Current=4, ForwardStack=[]
	if len(pc.ForwardStack) != 0 {
		t.Errorf("Next() should clear ForwardStack, got %d items", len(pc.ForwardStack))
	}
}
