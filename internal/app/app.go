package app

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
	"GO_player/internal/orchestrator"
	"GO_player/internal/storage"
)

func NewApp(dpPath string) (*orchestrator.Orchestrator, error) {
	db, err := storage.NewDB(dpPath)
	if err != nil {
		return nil, err
	}

	bg := basegraph.NewBaseGraph()
	edges, err := db.GetBaseGraph(0)
	if err != nil {
		return nil, err
	}
	err = bg.SetEdges(edges)
	if err != nil {
		return nil, err
	}

	s := selector.NewSelector()
	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(bg)

	return orchestrator.NewOrchestrator(bg, rg, s), nil
}
