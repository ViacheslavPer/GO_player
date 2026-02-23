package orchestrator

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
	"GO_player/internal/playback"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type lifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type runState int

const (
	stateRunning runState = iota
	stateShutDown
)

// closedChan — канал, уже закрытый при инициализации; возвращается из GetBGRebuildChan после Shutdown.
var closedChan = func() chan struct{} { c := make(chan struct{}); close(c); return c }()

type Orchestrator struct {
	baseGraph           *basegraph.BaseGraph
	rebuildChan         chan struct{}
	runtimeGraph        atomic.Pointer[runtime.RuntimeGraph]
	lifecycle           atomic.Pointer[lifecycle]
	maxRuntimeGraphAge  time.Duration
	maxRuntimeGraphDiff float64
	diffChan            chan struct{}
	selector            *selector.Selector
	playbackChain       *playback.PlaybackChain
	wg                  *sync.WaitGroup
	mu                  sync.RWMutex
	state               runState
}

func NewOrchestrator(bg *basegraph.BaseGraph, rg *runtime.RuntimeGraph, s *selector.Selector, pb *playback.PlaybackChain) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	if rg == nil {
		rg = runtime.NewRuntimeGraph()
		if bg != nil {
			rg.BuildFromBase(bg)
		}
	}
	if bg == nil {
		bg = basegraph.NewBaseGraph()
	}
	if s == nil {
		s = selector.NewSelector()
	}
	if pb == nil {
		pb = &playback.PlaybackChain{}
	}

	o := &Orchestrator{
		baseGraph:           bg,
		rebuildChan:         make(chan struct{}),
		maxRuntimeGraphAge:  time.Hour,
		maxRuntimeGraphDiff: 50.0,
		diffChan:            make(chan struct{}, 5),
		selector:            s,
		playbackChain:       pb,
		wg:                  wg,
		mu:                  sync.RWMutex{},
	}

	o.runtimeGraph.Store(rg)
	o.lifecycle.Store(&lifecycle{ctx: ctx, cancel: cancel})
	o.start()

	return o
}

func (o *Orchestrator) GetBaseGraph() *basegraph.BaseGraph {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.state == stateShutDown {
		return nil
	}
	return o.baseGraph
}

func (o *Orchestrator) GetPlayBackChain() *playback.PlaybackChain {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.state == stateShutDown {
		return nil
	}
	return o.playbackChain
}

func (o *Orchestrator) GetBGRebuildChan() <-chan struct{} {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.state == stateShutDown {
		return closedChan
	}
	return o.rebuildChan
}

func (o *Orchestrator) rebuildRuntime(rebuildReason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == stateShutDown {
		return
	}

	current := o.runtimeGraph.Load()
	if current == nil {
		return
	}
	runtimePenalty := current.GetPenalty()
	for fromID := range runtimePenalty {
		for toID := range runtimePenalty[fromID] {
			o.baseGraphPenalty(fromID, toID)
		}
	}

	o.stop()

	ctx, cancel := context.WithCancel(context.Background())
	o.lifecycle.Store(&lifecycle{ctx: ctx, cancel: cancel})

	newRG := runtime.NewRuntimeGraph()
	newRG.RebuildFromBase(o.baseGraph, rebuildReason)
	o.runtimeGraph.Store(newRG)

	o.start()
}

func (o *Orchestrator) start() {
	o.wg.Add(2)
	go o.manageRuntimeGraphTS()
	go o.manageRuntimeGraphDiffts()
}

func (o *Orchestrator) Shutdown() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == stateShutDown {
		return
	}
	o.stop()
	o.state = stateShutDown
}

func (o *Orchestrator) stop() {
	if h := o.lifecycle.Load(); h != nil {
		h.cancel()
	}
	o.wg.Wait()
	for {
		select {
		case <-o.diffChan:
		case <-o.rebuildChan:
		default:
			return
		}
	}
}

func (o *Orchestrator) manageRuntimeGraphTS() {
	defer o.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		lc := o.lifecycle.Load()
		if lc == nil {
			return
		}
		select {
		case <-lc.ctx.Done():
			return
		case <-ticker.C:
			rg := o.runtimeGraph.Load()
			if rg == nil {
				continue
			}

			runtimeGraphAge := time.Since(rg.GetTimestamp())
			if runtimeGraphAge > o.maxRuntimeGraphAge {
				go o.rebuildRuntime("time to live is up")
			}
		}
	}
}

func (o *Orchestrator) manageRuntimeGraphDiffts() {
	defer o.wg.Done()
	for {
		lc := o.lifecycle.Load()
		if lc == nil {
			return
		}
		select {
		case <-lc.ctx.Done():
			return
		case <-o.diffChan:
			rg := o.runtimeGraph.Load()
			if rg == nil {
				continue
			}

			runtimeGraphDiffts := rg.GetDiffts()
			if runtimeGraphDiffts > o.maxRuntimeGraphDiff {
				go o.rebuildRuntime("diff limit exceeded")
			}
		}
	}
}

func (o *Orchestrator) addChainSignal(chain chan struct{}) {
	lc := o.lifecycle.Load()
	if lc == nil {
		return
	}
	select {
	case <-lc.ctx.Done():
		return
	case chain <- struct{}{}:
		return
	}
}

func (o *Orchestrator) ProcessFeedbak(fromID, toID int64, listened, duration float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == stateShutDown {
		return
	}

	rg := o.runtimeGraph.Load()
	if rg == nil {
		return
	}

	progress := listened / duration
	if progress >= 0.33 {
		rg.Reinforce(fromID, toID, 1)
	} else if progress < 0.1 {
		rg.Penalty(fromID, toID, 2)
		rg.AddCooldown(fromID, toID, 0.2)
	} else {
		rg.Penalty(fromID, toID, 1)
		rg.AddCooldown(fromID, toID, 0.1)
	}
	o.addChainSignal(o.diffChan)
}

func (o *Orchestrator) baseGraphPenalty(fromID, toID int64) {
	o.baseGraph.Penalty(fromID, toID)
}

func (o *Orchestrator) PlayNext() (int64, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == stateShutDown {
		return 0, false
	}

	id, ok := o.playForward()
	if ok {
		return id, true
	}
	return o.generateNext()
}

func (o *Orchestrator) generateNext() (int64, bool) {
	var fromID int64
	if o.playbackChain.Current != 0 {
		fromID = o.playbackChain.Current
	}

	rg := o.runtimeGraph.Load()
	if rg == nil {
		return 0, false
	}

	toID, ok := o.selector.Next(fromID, rg)
	if !ok {
		return 0, false
	}

	o.playbackChain.Next(toID)

	return toID, true
}

func (o *Orchestrator) playForward() (int64, bool) {
	id, ok := o.playbackChain.Forward()
	if !ok {
		return 0, false
	}

	o.playbackChain.FreezeLearning()

	return id, true
}

func (o *Orchestrator) PlayBack() (int64, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == stateShutDown {
		return 0, false
	}

	id, ok := o.playbackChain.Back()
	if !ok {
		return 0, false
	}

	o.playbackChain.FreezeLearning()

	return id, true
}
