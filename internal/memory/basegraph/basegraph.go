package basegraph

import "sync"

type BaseGraph struct {
	mu    sync.RWMutex
	edges map[int64]map[int64]float64
}

func NewBaseGraph() *BaseGraph {
	return &BaseGraph{
		edges: make(map[int64]map[int64]float64),
	}
}

func (graph *BaseGraph) Reinforce(fromID, toID int64) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

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
	graph.mu.Lock()
	defer graph.mu.Unlock()

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

func (graph *BaseGraph) GetEdgesForID(id int64) map[int64]float64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	if graph.edges[id] == nil {
		return make(map[int64]float64)
	}

	src := graph.edges[id]
	copyMap := make(map[int64]float64, len(src))
	for k, v := range src {
		copyMap[k] = v
	}
	return copyMap
}

func (graph *BaseGraph) SetEdges(edges map[int64]map[int64]float64) error {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	if edges == nil {
		graph.edges = make(map[int64]map[int64]float64)
		return nil
	}

	newEdges := make(map[int64]map[int64]float64, len(edges))

	for id, neighbors := range edges {
		if neighbors == nil {
			newEdges[id] = make(map[int64]float64)
			continue
		}

		neighborCopy := make(map[int64]float64, len(neighbors))
		for k, v := range neighbors {
			neighborCopy[k] = v
		}

		newEdges[id] = neighborCopy
	}

	graph.edges = newEdges
	return nil
}

func (graph *BaseGraph) GetEdges() map[int64]map[int64]float64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	copyEdges := make(map[int64]map[int64]float64, len(graph.edges))

	for id, neighbors := range graph.edges {
		if neighbors == nil {
			copyEdges[id] = make(map[int64]float64)
			continue
		}

		neighborCopy := make(map[int64]float64, len(neighbors))
		for k, v := range neighbors {
			neighborCopy[k] = v
		}

		copyEdges[id] = neighborCopy
	}

	return copyEdges
}

func (graph *BaseGraph) GetAllIDs() []int64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	ids := make([]int64, 0, len(graph.edges))
	for id := range graph.edges {
		ids = append(ids, id)
	}
	return ids
}
