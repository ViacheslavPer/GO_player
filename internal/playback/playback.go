package playback

// PlaybackChain manages back/forward navigation.
// Does NOT know about memory or graphs.
type PlaybackChain struct {
	BackStack      []int64 //хранить историю предыдущих треков
	Current        int64   //текущий трек
	ForwardStack   []int64 //хранить историю вперед после_Back()
	LearningFrozen bool    //заморозка обучения
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
	pc.LearningFrozen = true
}

func (pc *PlaybackChain) UnfreezeLearning() {
	pc.LearningFrozen = false
}
