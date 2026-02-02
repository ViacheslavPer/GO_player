package runtime

import (
	"GO_player/internal/memory/basegraph"
	"time"
)

type RuntimeGraph struct {
	edges        map[int64]map[int64]int64
	cooldowns    map[int64]map[int64]int64
	penalties    map[int64]map[int64]int64
	buildVersion int64
	buildReason  string
	timestamp    int64
}

func NewRuntimeGraph() *RuntimeGraph {
	return &RuntimeGraph{
		edges:     make(map[int64]map[int64]int64),
		cooldowns: make(map[int64]map[int64]int64),
		penalties: make(map[int64]map[int64]int64),
	}
}

func (graph *RuntimeGraph) GetBuildVersion() int64 {
	return graph.buildVersion
}

func (graph *RuntimeGraph) GetBuildReason() string {
	return graph.buildReason
}

func (graph *RuntimeGraph) GetTimestamp() int64 {
	return graph.timestamp
}

func (graph *RuntimeGraph) AddCooldown(fromID int64, toID int64, value int64) {
	if graph.cooldowns[fromID] == nil {
		graph.cooldowns[fromID] = make(map[int64]int64)
	}
	graph.cooldowns[fromID][toID] = value
}

func (graph *RuntimeGraph) ReduceCooldown() {
	for _, inner := range graph.cooldowns {
		for toID := range inner {
			if inner[toID] > 0 {
				inner[toID]--
			}
		}
	}
}

func (graph *RuntimeGraph) AddPenalty(fromID int64, toID int64, value int64) {
	if graph.penalties[fromID] == nil {
		graph.penalties[fromID] = make(map[int64]int64)
	}
	graph.penalties[fromID][toID] = value
}

func (graph *RuntimeGraph) CopyBase(base *basegraph.BaseGraph, buildVersion int64, buildReason string) {
	graph.edges = make(map[int64]map[int64]int64)

	ids := base.GetAllIDs()
	for _, fid := range ids {
		baseStat := base.GetEdges(fid)
		if baseStat == nil || len(baseStat) == 0 {
			continue
		}

		if graph.edges[fid] == nil {
			graph.edges[fid] = make(map[int64]int64, len(baseStat))
		}

		for sid, weight := range baseStat {
			graph.edges[fid][sid] = weight
		}
	}

	graph.buildReason = buildReason
	graph.buildVersion = buildVersion
	graph.timestamp = time.Now().Unix()
}

func copyMap[K comparable, V any](src map[K]V) map[K]V {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (graph *RuntimeGraph) calculateFines(fromID int64) map[int64]int64 {
	if graph.edges[fromID] == nil {
		return map[int64]int64{}
	}
	if graph.cooldowns[fromID] == nil && graph.penalties[fromID] == nil {
		return copyMap(graph.edges[fromID])
	}

	fined := copyMap(graph.edges[fromID])

	if graph.cooldowns[fromID] != nil {
		for toID := range fined {
			if cd, ok := graph.cooldowns[fromID][toID]; ok && fined[toID] > cd {
				fined[toID] -= cd
			}
		}
	}

	if graph.penalties[fromID] != nil {
		for toID := range fined {
			if cd, ok := graph.penalties[fromID][toID]; ok && fined[toID] > cd {
				fined[toID] -= cd
			}
		}
	}

	return fined
}

func calculateProb(fined map[int64]int64) map[int64]float64 {
	if fined == nil {
		return make(map[int64]float64)
	}

	prob := make(map[int64]float64)

	sum := int64(0)
	for _, value := range fined {
		sum += value
	}
	if sum == 0 {
		return make(map[int64]float64)
	}

	for id, value := range fined {
		prob[id] = float64(value) / float64(sum)
	}

	return prob
}

func (graph *RuntimeGraph) GetEdges(fromID int64) map[int64]float64 {
	fined := graph.calculateFines(fromID)
	if len(fined) == 0 {
		return map[int64]float64{}
	}
	return calculateProb(fined)
}
