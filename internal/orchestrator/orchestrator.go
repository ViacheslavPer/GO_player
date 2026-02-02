package orchestrator

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
)

type Orchestrator struct {
	baseGraph    *basegraph.BaseGraph
	runtimeGraph *runtime.RuntimeGraph
	selector     *selector.Selector
}

func NewOrchestrator() *Orchestrator {
	bg := basegraph.NewBaseGraph()

	rg := runtime.NewRuntimeGraph()
	rg.CopyBase(bg, 0, "")

	s := selector.NewSelector()

	return &Orchestrator{
		baseGraph:    bg,
		runtimeGraph: rg,
		selector:     s,
	}
}

func (o *Orchestrator) Learn(fromID, toID int64) {
	o.baseGraph.Reinforce(fromID, toID)
}

func (o *Orchestrator) Penalize(fromID, toID int64) {
	o.baseGraph.Penalty(fromID, toID)
}

func (o *Orchestrator) RebuildRuntime() {
	o.runtimeGraph = runtime.NewRuntimeGraph()
	o.runtimeGraph.CopyBase(o.baseGraph, 0, "")
}

func (o *Orchestrator) Next(fromID int64) (toID int64, ok bool) {
	return o.selector.Next(fromID, o.runtimeGraph)
}

func (o *Orchestrator) GetRuntimeGraph() *runtime.RuntimeGraph {
	return o.runtimeGraph
}
