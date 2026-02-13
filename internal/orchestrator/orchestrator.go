package orchestrator

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
	"GO_player/internal/playback"
	"context"
	"sync"
	"time"
)

type Orchestrator struct {
	baseGraph           *basegraph.BaseGraph
	runtimeGraph        *runtime.RuntimeGraph
	maxRuntimeGraphAge  time.Duration
	maxRuntimeGraphDiff float64
	diffChan            chan struct{}
	runtimeGraphVersion int64
	runtimeGraphMutex   sync.RWMutex
	selector            *selector.Selector
	playbackChain       *playback.PlaybackChain
	playbackMutex       sync.Mutex
	cooldownChan        chan struct{}
	wg                  *sync.WaitGroup
	ctx                 context.Context
	cancel              context.CancelFunc
}

func NewOrchestrator() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	bg := basegraph.NewBaseGraph()

	rg := runtime.NewRuntimeGraph()
	rg.CopyBase(bg, 0, "Orchestrator initialization")

	s := selector.NewSelector()

	return &Orchestrator{
		baseGraph:           bg,
		runtimeGraph:        rg,
		maxRuntimeGraphAge:  time.Hour,
		maxRuntimeGraphDiff: 50,
		diffChan:            make(chan struct{}),
		runtimeGraphMutex:   sync.RWMutex{},
		selector:            s,
		playbackChain:       &playback.PlaybackChain{},
		playbackMutex:       sync.Mutex{},
		cooldownChan:        make(chan struct{}),
		wg:                  wg,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

func (o *Orchestrator) RebuildRuntime(rebuildReason string) {
	o.runtimeGraphMutex.Lock()
	defer o.runtimeGraphMutex.Unlock()
	runtimePenalty := o.runtimeGraph.GetPenalty()
	for fromID := range runtimePenalty {
		for toID := range runtimePenalty[fromID] {
			o.baseGraph.Penalty(fromID, toID)
		}
	}
	o.runtimeGraphVersion++
	o.runtimeGraph = runtime.NewRuntimeGraph()
	o.runtimeGraph.CopyBase(o.baseGraph, o.runtimeGraphVersion, rebuildReason)
}

func (o *Orchestrator) Start() {
	o.wg.Add(3)
	go o.manageCooldowns()
	go o.manageRuntimeGraphTS()
	go o.manageRuntimeGraphDiffts()
}

func (o *Orchestrator) Stop() {
	o.cancel()
	o.wg.Wait()
}

func (o *Orchestrator) manageCooldowns() {
	defer o.wg.Done()
	for {
		select {
		case <-o.ctx.Done():
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
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			runtimeGraphAge := time.Since(o.runtimeGraph.GetTimestamp())
			if runtimeGraphAge > o.maxRuntimeGraphAge {
				o.RebuildRuntime("time to live is up")
			}
		}
	}
}

func (o *Orchestrator) manageRuntimeGraphDiffts() {
	defer o.wg.Done()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-o.diffChan:
			runtimeGraphDiffts := o.runtimeGraph.GetDiffts()
			if runtimeGraphDiffts > o.maxRuntimeGraphDiff {
				o.RebuildRuntime("diff limit exceeded")
			}
		}
	}
}

func (o *Orchestrator) Learn(fromID, toID int64) {
	o.cooldownChan <- struct{}{}
	if !o.playbackChain.LearningFrozen {
		o.baseGraph.Reinforce(fromID, toID)
		o.runtimeGraph.Reinforce(fromID, toID)
	}
}

func (o *Orchestrator) baseGraphPenalty(fromID, toID int64) {
	o.baseGraph.Penalty(fromID, toID)
}

func (o *Orchestrator) addCooldown(fromID, toID int64, value float64) {
	if value <= 0.0 {
		value = 1.0
	}
	o.runtimeGraph.AddCooldown(fromID, toID, value)
}

func (o *Orchestrator) reduceCooldown() {
	o.runtimeGraph.ReduceCooldown()
}

func (o *Orchestrator) runtimeGraphPenalty(fromID, toID int64) {
	o.runtimeGraph.Penalty(fromID, toID)
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

	toID, ok := o.selector.Next(fromID, o.runtimeGraph)
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
	o.runtimeGraphMutex.RLock()
	defer o.runtimeGraphMutex.RUnlock()
	return o.runtimeGraph
}

func (o *Orchestrator) PlaybackChain() *playback.PlaybackChain {
	o.playbackMutex.Lock()
	defer o.playbackMutex.Unlock()
	return o.playbackChain
}
