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
	cooldownChan        chan struct{}
	wg                  *sync.WaitGroup
	rebuildMu           sync.Mutex
}

func NewOrchestrator() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	bg := basegraph.NewBaseGraph()

	s := selector.NewSelector()

	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(bg)

	o := &Orchestrator{
		baseGraph:           bg,
		maxRuntimeGraphAge:  time.Hour,
		maxRuntimeGraphDiff: 50,
		diffChan:            make(chan struct{}, 5),
		selector:            s,
		playbackChain:       &playback.PlaybackChain{},
		playbackMutex:       sync.Mutex{},
		cooldownChan:        make(chan struct{}, 1),
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
	// Keep existing diffChan and cooldownChan so in-flight addChainSignal goroutines
	// do not block forever on a channel no one receives from.

	newRG := runtime.NewRuntimeGraph()
	newRG.RebuildFromBase(o.baseGraph, rebuildReason)
	o.runtimeGraph.Store(newRG)

	o.Start()
}

func (o *Orchestrator) Start() {
	o.wg.Add(3)
	go o.manageCooldowns()
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

func (o *Orchestrator) manageCooldowns() {
	defer o.wg.Done()
	for {
		lc := o.lifecycle.Load()
		if lc == nil {
			return
		}
		select {
		case <-lc.ctx.Done():
			return
		case <-o.cooldownChan:
			o.reduceCooldown()
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

func (o *Orchestrator) Learn(fromID, toID int64) {
	go o.addChainSignal(o.cooldownChan)

	o.playbackMutex.Lock()
	learningFrozen := o.playbackChain.LearningFrozen
	o.playbackMutex.Unlock()

	if learningFrozen {
		return
	}

	o.baseGraph.Reinforce(fromID, toID)

	rg := o.runtimeGraph.Load()
	if rg != nil {
		rg.Reinforce(fromID, toID)
		go o.addChainSignal(o.diffChan)
	}
}

func (o *Orchestrator) baseGraphPenalty(fromID, toID int64) {
	o.baseGraph.Penalty(fromID, toID)
}

func (o *Orchestrator) addCooldown(fromID, toID int64, value float64) {
	if value <= 0.0 {
		value = 1.0
	}
	rg := o.runtimeGraph.Load()
	if rg != nil {
		rg.AddCooldown(fromID, toID, value)
	}
}

func (o *Orchestrator) reduceCooldown() {
	rg := o.runtimeGraph.Load()
	if rg != nil {
		rg.ReduceCooldown()
	}
}

func (o *Orchestrator) runtimeGraphPenalty(fromID, toID int64) {
	rg := o.runtimeGraph.Load()
	if rg != nil {
		rg.Penalty(fromID, toID)
		go o.addChainSignal(o.diffChan)
	}
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
