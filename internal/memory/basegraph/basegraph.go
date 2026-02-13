package basegraph

// BaseGraph stores long-term transition memory.
// It is a pure data structure with integer weights.
// No probabilities, no runtime logic, no UX logic.
type BaseGraph struct {
	edges map[int64]map[int64]float64
}

func NewBaseGraph() *BaseGraph {
	return &BaseGraph{
		edges: make(map[int64]map[int64]float64),
	}
}

func (graph *BaseGraph) Reinforce(fromID, toID int64) {
	if graph.edges[fromID] == nil {
		graph.edges[fromID] = make(map[int64]float64)
	}
	if graph.edges[0] == nil {
		graph.edges[0] = make(map[int64]float64)
	}
	graph.edges[0][toID]++
	graph.edges[fromID][toID]++
}

func (graph *BaseGraph) Penalty(fromID, toID int64) {
	if graph.edges[fromID] == nil {
		return
	}
	if graph.edges[fromID][toID] > 0 {
		graph.edges[fromID][toID]--
	}
	if graph.edges[0] != nil {
		if graph.edges[0][toID] > 0 {
			graph.edges[0][toID]--
		}
	}
}

func (graph *BaseGraph) GetEdges(id int64) map[int64]float64 {
	if graph.edges[id] == nil {
		return make(map[int64]float64)
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
