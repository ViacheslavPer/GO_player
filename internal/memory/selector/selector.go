package selector

import (
	"GO_player/internal/memory/runtime"
	"math"
	"math/rand"
	"sort"
)

type Selector struct {
	giniHigh float64
	giniLow  float64
	topK     int64
}

func NewSelector() *Selector {
	return &Selector{
		giniHigh: 0.6,
		giniLow:  0.35,
		topK:     10,
	}
}

func NewSelectorWithParameters(giniHigh, giniLow float64, topK int64) *Selector {
	if topK <= 0 {
		topK = 10
	}
	if giniLow <= 0.0 || giniLow >= 1.0 {
		giniLow = 0.35
	}
	if giniHigh <= 0.0 || giniHigh >= 1.0 {
		giniHigh = 0.6
	}
	if giniHigh <= giniLow {
		giniHigh = 0.6
		giniLow = 0.35
	}

	return &Selector{
		giniHigh: giniHigh,
		giniLow:  giniLow,
		topK:     topK,
	}
}

func computeTopK(N int, ratio float64) int {
	K_min := max(3, int(math.Ceil(float64(N)*0.05)))
	K_max := max(K_min+1, int(math.Ceil(float64(N)*0.3)))
	K := int(math.Round(float64(K_min) + ratio*float64(K_max-K_min)))
	if K < K_min {
		K = K_min
	}
	if K > K_max {
		K = K_max
	}
	return K
}

func computeGini(probs map[int64]float64) float64 {
	var sumSquares = 0.0

	for _, p := range probs {
		sumSquares += p * p
	}

	gini := 1.0 - sumSquares
	return gini
}

func (s *Selector) Next(fromID int64, runtimeGraph *runtime.RuntimeGraph) (toID int64, ok bool) {
	probs := runtimeGraph.GetEdges(fromID)
	if len(probs) == 0 {
		return 0, false
	}
	gini := computeGini(probs)

	ratio := (gini - s.giniLow) / (s.giniHigh - s.giniLow)
	ratio = math.Max(0.0, math.Min(1.0, ratio))

	if gini <= s.giniLow {
		return selectWeighted(probs)
	}

	alpha := 1.1 + (1.0-ratio)*0.7
	hybridProbs := make(map[int64]float64, len(probs))
	sum := 0.0
	for id, p := range probs {
		hybridProbs[id] = math.Pow(p, alpha)
		sum += hybridProbs[id]
	}
	for id := range hybridProbs {
		hybridProbs[id] /= sum
	}

	if gini >= s.giniHigh {
		k := computeTopK(len(probs), ratio)
		return selectTopK(hybridProbs, k)
	}

	return selectWeighted(hybridProbs)
}

func selectTopK(probs map[int64]float64, k int) (int64, bool) {
	type probItem struct {
		id   int64
		prob float64
	}

	items := make([]probItem, 0, len(probs))
	for id, p := range probs {
		items = append(items, probItem{id, p})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].prob > items[j].prob
	})

	k = int(math.Min(float64(k), float64(len(items))))

	topK := items[:k]
	idx := rand.Intn(len(topK))
	return topK[idx].id, true
}

func selectWeighted(probs map[int64]float64) (int64, bool) {
	type probItem struct {
		id   int64
		prob float64
	}
	items := make([]probItem, 0, len(probs))
	for id, p := range probs {
		items = append(items, probItem{id, p})
	}

	f := rand.Float64()

	sum := 0.0
	for _, item := range items {
		sum += item.prob
		if f < sum {
			return item.id, true
		}
	}
	return items[len(items)-1].id, true
}
