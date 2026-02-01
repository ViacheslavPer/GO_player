package basegraph

// BaseGraph stores long-term transition memory.
// It is a pure data structure with integer weights.
// No probabilities, no runtime logic, no UX logic.
type BaseGraph struct {
	edges map[int64]map[int64]int64
}

func NewBaseGraph() *BaseGraph {
	return &BaseGraph{
		edges: make(map[int64]map[int64]int64),
	}
}

func (graph *BaseGraph) Reinforce(fid, sid int64) {
	if graph.edges[fid] == nil {
		graph.edges[fid] = make(map[int64]int64)
	}
	graph.edges[fid][sid]++
}

func (graph *BaseGraph) Penalty(fid, sid int64) {
	if graph.edges[fid] == nil {
		return
	}
	if graph.edges[fid][sid] > 0 {
		graph.edges[fid][sid]--
	}
}

func (graph *BaseGraph) GetEdges(id int64) map[int64]int64 {
	if graph.edges[id] == nil {
		return make(map[int64]int64)
	}
	return graph.edges[id]
}

func (graph *BaseGraph) GetAllIDs() []int64 {
	ids := make([]int64, 0, len(graph.edges))
	for id := range graph.edges {
		ids = append(ids, id)
	}
	return ids
}
