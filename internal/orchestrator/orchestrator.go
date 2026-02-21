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

type Orchestrator struct {
	baseGraph           *basegraph.BaseGraph
	runtimeGraph        atomic.Pointer[runtime.RuntimeGraph]
	lifecycle           atomic.Pointer[lifecycle]
	maxRuntimeGraphAge  time.Duration
	maxRuntimeGraphDiff float64
	diffChan            chan struct{}
	selector            *selector.Selector
	playbackChain       *playback.PlaybackChain
	playbackMutex       sync.Mutex
	wg                  *sync.WaitGroup
	rebuildMu           sync.Mutex
}

func NewOrchestrator(bg *basegraph.BaseGraph, rg *runtime.RuntimeGraph, s *selector.Selector) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	o := &Orchestrator{
		baseGraph:           bg,
		maxRuntimeGraphAge:  time.Hour,
		maxRuntimeGraphDiff: 50.0,
		diffChan:            make(chan struct{}, 5),
		selector:            s,
		playbackChain:       &playback.PlaybackChain{},
		playbackMutex:       sync.Mutex{},
		wg:                  wg,
		rebuildMu:           sync.Mutex{},
	}

	o.runtimeGraph.Store(rg)
	o.lifecycle.Store(&lifecycle{ctx: ctx, cancel: cancel})

	return o
}

func (o *Orchestrator) RebuildRuntime(rebuildReason string) {
	o.rebuildMu.Lock()
	defer o.rebuildMu.Unlock()

	current := o.runtimeGraph.Load()
	if current == nil {
		return
	}
	runtimePenalty := current.GetPenalty()
	for fromID := range runtimePenalty {
		for toID := range runtimePenalty[fromID] {
			o.baseGraph.Penalty(fromID, toID)
		}
	}

	o.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	o.lifecycle.Store(&lifecycle{ctx: ctx, cancel: cancel})

	newRG := runtime.NewRuntimeGraph()
	newRG.RebuildFromBase(o.baseGraph, rebuildReason)
	o.runtimeGraph.Store(newRG)

	o.Start()
}

func (o *Orchestrator) Start() {
	o.wg.Add(2)
	go o.manageRuntimeGraphTS()
	go o.manageRuntimeGraphDiffts()
}

func (o *Orchestrator) Stop() {
	o.rebuildMu.Lock()
	defer o.rebuildMu.Unlock()
	o.stopLocked()
}

func (o *Orchestrator) stopLocked() {
	if h := o.lifecycle.Load(); h != nil {
		h.cancel()
	}
	o.wg.Wait()
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
				go o.RebuildRuntime("time to live is up")
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
				go o.RebuildRuntime("diff limit exceeded")
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
	id, ok := o.playForward()
	if ok {
		return id, true
	}
	return o.generateNext()
}

func (o *Orchestrator) generateNext() (int64, bool) {
	o.playbackMutex.Lock()
	defer o.playbackMutex.Unlock()

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
	o.playbackMutex.Lock()
	defer o.playbackMutex.Unlock()

	id, ok := o.playbackChain.Forward()
	if !ok {
		return 0, false
	}

	o.playbackChain.FreezeLearning()

	return id, true
}

func (o *Orchestrator) PlayBack() (int64, bool) {
	o.playbackMutex.Lock()
	defer o.playbackMutex.Unlock()

	id, ok := o.playbackChain.Back()
	if !ok {
		return 0, false
	}

	o.playbackChain.FreezeLearning()

	return id, true
}

// Getter methods for testing (read-only access to internal state)
func (o *Orchestrator) BaseGraph() *basegraph.BaseGraph {
	return o.baseGraph
}

func (o *Orchestrator) RuntimeGraph() *runtime.RuntimeGraph {
	return o.runtimeGraph.Load()
}

func (o *Orchestrator) PlaybackChain() *playback.PlaybackChain {
	o.playbackMutex.Lock()
	defer o.playbackMutex.Unlock()
	return o.playbackChain
}
