package playback

import "testing"

func TestPlaybackChain_NextBackForward(t *testing.T) {
	var pc PlaybackChain

	if _, ok := pc.Back(); ok {
		t.Fatalf("Back() on empty chain ok=true, want false")
	}
	if _, ok := pc.Forward(); ok {
		t.Fatalf("Forward() on empty chain ok=true, want false")
	}

	if got, ok := pc.Next(1); !ok || got != 1 {
		t.Fatalf("Next(1) = (%d,%v), want (1,true)", got, ok)
	}
	if pc.Current != 1 || len(pc.BackStack) != 0 || len(pc.ForwardStack) != 0 {
		t.Fatalf("unexpected state after Next(1): %+v", pc)
	}

	if got, ok := pc.Next(2); !ok || got != 2 {
		t.Fatalf("Next(2) = (%d,%v), want (2,true)", got, ok)
	}
	if pc.Current != 2 || len(pc.BackStack) != 1 || pc.BackStack[0] != 1 || len(pc.ForwardStack) != 0 {
		t.Fatalf("unexpected state after Next(2): %+v", pc)
	}

	if got, ok := pc.Back(); !ok || got != 1 {
		t.Fatalf("Back() = (%d,%v), want (1,true)", got, ok)
	}
	if pc.Current != 1 || len(pc.BackStack) != 0 || len(pc.ForwardStack) != 1 || pc.ForwardStack[0] != 2 {
		t.Fatalf("unexpected state after Back(): %+v", pc)
	}

	if got, ok := pc.Forward(); !ok || got != 2 {
		t.Fatalf("Forward() = (%d,%v), want (2,true)", got, ok)
	}
	if pc.Current != 2 || len(pc.BackStack) != 1 || pc.BackStack[0] != 1 || len(pc.ForwardStack) != 0 {
		t.Fatalf("unexpected state after Forward(): %+v", pc)
	}

	// Next clears forward stack.
	pc.ForwardStack = append(pc.ForwardStack, 999)
	if got, ok := pc.Next(3); !ok || got != 3 {
		t.Fatalf("Next(3) = (%d,%v), want (3,true)", got, ok)
	}
	if pc.Current != 3 || len(pc.ForwardStack) != 0 {
		t.Fatalf("forward stack not cleared on Next(): %+v", pc)
	}
}

func TestPlaybackChain_FreezeUnfreezeLearning_Idempotent(t *testing.T) {
	var pc PlaybackChain

	if pc.LearningFrozen {
		t.Fatalf("LearningFrozen initially true, want false")
	}
	pc.FreezeLearning()
	if !pc.LearningFrozen {
		t.Fatalf("FreezeLearning did not freeze")
	}
	pc.FreezeLearning() // idempotent
	if !pc.LearningFrozen {
		t.Fatalf("FreezeLearning not idempotent")
	}

	pc.UnfreezeLearning()
	if pc.LearningFrozen {
		t.Fatalf("UnfreezeLearning did not unfreeze")
	}
	pc.UnfreezeLearning() // idempotent
	if pc.LearningFrozen {
		t.Fatalf("UnfreezeLearning not idempotent")
	}
}
