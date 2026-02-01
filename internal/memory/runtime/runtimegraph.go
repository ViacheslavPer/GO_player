package runtime

import "GO_player/internal/memory/basegraph"

type RuntimeGraph struct {
	edges map[int64]map[int64]float64
}

func NewRuntimeGraph() *RuntimeGraph {
	return &RuntimeGraph{edges: make(map[int64]map[int64]float64)}
}

func (graph *RuntimeGraph) BuildFromBase(base *basegraph.BaseGraph) {
	graph.edges = make(map[int64]map[int64]float64)

	ids := base.GetAllIDs()
	for _, fid := range ids {
		baseStat := base.GetEdges(fid)

		var sum int64 = 0
		for _, weight := range baseStat {
			sum += weight
		}
		if sum == 0 {
			continue
		}
		for sid, weight := range baseStat {
			if graph.edges[fid] == nil {
				graph.edges[fid] = make(map[int64]float64)
			}
			graph.edges[fid][sid] = float64(weight) / float64(sum)
		}
	}
}

func (graph *RuntimeGraph) GetEdges(id int64) map[int64]float64 {
	if graph.edges[id] == nil {
		return make(map[int64]float64)
	}
	return graph.edges[id]
}
