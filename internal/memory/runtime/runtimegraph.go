package runtime

import (
	"GO_player/internal/memory/basegraph"
	"math"
	"sync"
	"time"
)

type cooldownEntry struct {
	Value float64
	Ts    time.Time
}
type RuntimeGraph struct {
	mu           sync.RWMutex
	edges        map[int64]map[int64]float64
	cooldowns    map[int64]map[int64]cooldownEntry
	tau          float64
	penalties    map[int64]map[int64]float64
	buildVersion int64
	buildReason  string
	timestamp    time.Time
	diffts       float64
}

func NewRuntimeGraph() *RuntimeGraph {
	return &RuntimeGraph{
		edges:     make(map[int64]map[int64]float64),
		cooldowns: make(map[int64]map[int64]cooldownEntry),
		penalties: make(map[int64]map[int64]float64),
		tau:       180,
	}
}

func (graph *RuntimeGraph) GetBuildVersion() int64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()
	return graph.buildVersion
}

func (graph *RuntimeGraph) GetBuildReason() string {
	graph.mu.RLock()
	defer graph.mu.RUnlock()
	return graph.buildReason
}

func (graph *RuntimeGraph) GetTimestamp() time.Time {
	graph.mu.RLock()
	defer graph.mu.RUnlock()
	return graph.timestamp
}

func (graph *RuntimeGraph) GetDiffts() float64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()
	return graph.diffts
}

func (graph *RuntimeGraph) GetPenalty() map[int64]map[int64]float64 {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	copyOuter := make(map[int64]map[int64]float64, len(graph.penalties))
	for fromID, inner := range graph.penalties {
		copyOuter[fromID] = copyMap(inner) // ← вот тут ок
	}
	return copyOuter
}

func (graph *RuntimeGraph) Reinforce(fromID, toID int64) {
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
	graph.diffts++
}

func (graph *RuntimeGraph) AddCooldown(fromID, toID int64, value float64) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	if graph.cooldowns[fromID] == nil {
		graph.cooldowns[fromID] = make(map[int64]cooldownEntry)
	}

	if value <= 0.0 || value >= 1.0 {
		value = 1.0
	}

	graph.cooldowns[fromID][toID] = cooldownEntry{
		Value: value,
		Ts:    time.Now(),
	}
	graph.diffts++
}

func (graph *RuntimeGraph) Penalty(fromID, toID int64) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	if graph.penalties[fromID] == nil {
		graph.penalties[fromID] = make(map[int64]float64)
	}
	graph.penalties[fromID][toID]++
	graph.diffts++
}

func (graph *RuntimeGraph) BuildFromBase(base *basegraph.BaseGraph) {
	graph.mu.Lock()
	defer graph.mu.Unlock()
	graph.copyBase(base, 0, "Runtime initialization")
}

func (graph *RuntimeGraph) RebuildFromBase(base *basegraph.BaseGraph, buildReason string) {
	graph.mu.Lock()
	defer graph.mu.Unlock()
	graph.buildVersion++
	graph.copyBase(base, graph.buildVersion, buildReason)
}

// CopyBase is an exported, concurrency-safe wrapper used primarily in tests.
// It delegates to the internal copyBase implementation under the graph mutex.
func (graph *RuntimeGraph) CopyBase(base *basegraph.BaseGraph, buildVersion int64, buildReason string) {
	graph.mu.Lock()
	defer graph.mu.Unlock()
	graph.copyBase(base, buildVersion, buildReason)
}

func (graph *RuntimeGraph) copyBase(base *basegraph.BaseGraph, buildVersion int64, buildReason string) {
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
	if graph.edges[fromID] == nil {
		return map[int64]float64{}
	}
	if graph.cooldowns[fromID] == nil && graph.penalties[fromID] == nil {
		return copyMap(graph.edges[fromID])
	}

	fined := copyMap(graph.edges[fromID])

	if graph.cooldowns[fromID] != nil {
		for toID := range fined {
			if cd, ok := graph.cooldowns[fromID][toID]; ok {
				cooldownScore := cd.Value * math.Exp(-time.Since(cd.Ts).Seconds()/graph.tau)
				if fined[toID] > cooldownScore {
					fined[toID] -= cooldownScore
				} else {
					fined[toID] = 0
				}
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
	graph.mu.RLock()
	defer graph.mu.RUnlock()
	fined := graph.calculateFines(fromID)
	if len(fined) == 0 {
		return map[int64]float64{}
	}
	return calculateProb(fined)
}
