package runtime

import (
	"GO_player/internal/memory/basegraph"
	"sync"
	"time"
)

type RuntimeGraph struct {
	edges        map[int64]map[int64]float64
	edMu         sync.RWMutex
	cooldowns    map[int64]map[int64]float64
	cdMu         sync.RWMutex
	penalties    map[int64]map[int64]float64
	pMu          sync.RWMutex
	buildVersion int64
	buildReason  string
	timestamp    time.Time
	diffts       float64
}

func NewRuntimeGraph() *RuntimeGraph {
	return &RuntimeGraph{
		edges:     make(map[int64]map[int64]float64),
		cooldowns: make(map[int64]map[int64]float64),
		penalties: make(map[int64]map[int64]float64),
	}
}

func (graph *RuntimeGraph) GetBuildVersion() int64 {
	return graph.buildVersion
}

func (graph *RuntimeGraph) GetBuildReason() string {
	return graph.buildReason
}

func (graph *RuntimeGraph) GetTimestamp() time.Time {
	return graph.timestamp
}

func (graph *RuntimeGraph) GetDiffts() float64 { return graph.diffts }

func (graph *RuntimeGraph) GetPenalty() map[int64]map[int64]float64 { return graph.penalties }

func (graph *RuntimeGraph) Reinforce(fromID, toID int64) {
	if graph.edges[fromID] == nil {
		graph.edges[fromID] = make(map[int64]float64)
	}
	if graph.edges[0] == nil {
		graph.edges[0] = make(map[int64]float64)
	}
	graph.edges[0][toID]++
	graph.edges[fromID][toID]++
}

func (graph *RuntimeGraph) AddCooldown(fromID, toID int64, value float64) {
	graph.cdMu.Lock()
	defer graph.cdMu.Unlock()

	if graph.cooldowns[fromID] == nil {
		graph.cooldowns[fromID] = make(map[int64]float64)
	}
	graph.cooldowns[fromID][toID] = value
	graph.diffts++
}

func (graph *RuntimeGraph) ReduceCooldown() {
	graph.cdMu.Lock()
	defer graph.cdMu.Unlock()

	for _, inner := range graph.cooldowns {
		for toID := range inner {
			if inner[toID] > 0 {
				inner[toID]--
			}
			if graph.diffts > 0 {
				graph.diffts--
			}
		}
	}
}

func (graph *RuntimeGraph) Penalty(fromID, toID int64) {
	graph.pMu.Lock()
	defer graph.pMu.Unlock()

	if graph.penalties[fromID] == nil {
		graph.penalties[fromID] = make(map[int64]float64)
	}
	graph.penalties[fromID][toID]++
	graph.diffts++
}

func (graph *RuntimeGraph) CopyBase(base *basegraph.BaseGraph, buildVersion int64, buildReason string) {
	graph.edMu.Lock()
	defer graph.edMu.Unlock()

	graph.edges = make(map[int64]map[int64]float64)

	ids := base.GetAllIDs()
	for _, fid := range ids {
		baseStat := base.GetEdges(fid)
		if baseStat == nil || len(baseStat) == 0 {
			continue
		}

		if graph.edges[fid] == nil {
			graph.edges[fid] = make(map[int64]float64, len(baseStat))
		}

		for sid, weight := range baseStat {
			graph.edges[fid][sid] = weight
		}
	}

	graph.buildReason = buildReason
	graph.buildVersion = buildVersion
	graph.timestamp = time.Now()
	graph.diffts = 0.0
}

func copyMap[K comparable, V any](src map[K]V) map[K]V {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (graph *RuntimeGraph) calculateFines(fromID int64) map[int64]float64 {
	graph.edMu.RLock()
	graph.cdMu.RLock()
	graph.pMu.RLock()
	defer graph.edMu.RUnlock()
	defer graph.cdMu.RUnlock()
	defer graph.pMu.RUnlock()

	if graph.edges[fromID] == nil {
		return map[int64]float64{}
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

func calculateProb(fined map[int64]float64) map[int64]float64 {
	if fined == nil {
		return make(map[int64]float64)
	}

	prob := make(map[int64]float64)

	sum := 0.0
	for _, value := range fined {
		sum += value
	}
	if sum == 0 {
		return make(map[int64]float64)
	}

	for id, value := range fined {
		prob[id] = value / sum
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
