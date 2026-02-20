package playback

type PlaybackChain struct {
	BackStack      []int64 `json:"back_stack"`
	Current        int64   `json:"current"`
	ForwardStack   []int64 `json:"forward_stack"`
	LearningFrozen bool    `json:"learning_frozen"`
}

func (pc *PlaybackChain) Back() (int64, bool) {
	if len(pc.BackStack) == 0 || pc.Current == 0 {
		return 0, false
	}
	pc.ForwardStack = append(pc.ForwardStack, pc.Current)
	pc.Current = pc.BackStack[len(pc.BackStack)-1]
	pc.BackStack = pc.BackStack[:len(pc.BackStack)-1]
	return pc.Current, true
}

func (pc *PlaybackChain) Next(id int64) (int64, bool) {
	if pc.Current != 0 {
		pc.BackStack = append(pc.BackStack, pc.Current)
	}
	pc.Current = id
	pc.ForwardStack = pc.ForwardStack[:0]
	return pc.Current, true
}

func (pc *PlaybackChain) Forward() (int64, bool) {
	if len(pc.ForwardStack) != 0 {
		if pc.Current != 0 {
			pc.BackStack = append(pc.BackStack, pc.Current)
		}
		pc.Current = pc.ForwardStack[len(pc.ForwardStack)-1]
		pc.ForwardStack = pc.ForwardStack[:len(pc.ForwardStack)-1]
		return pc.Current, true
	}

	return 0, false
}

func (pc *PlaybackChain) FreezeLearning() {
	if pc.LearningFrozen {
		return
	}
	pc.LearningFrozen = true
}

func (pc *PlaybackChain) UnfreezeLearning() {
	if !pc.LearningFrozen {
		return
	}
	pc.LearningFrozen = false
}
