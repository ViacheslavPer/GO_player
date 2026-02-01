package selector

import (
	"GO_player/internal/memory/runtime"
	"math"
	"math/rand"
	"sort"
)

type Selector struct {
	giniThreshold float64
	topK          int64
}

func NewSelector() *Selector {
	return &Selector{
		giniThreshold: 0.5,
		topK:          10,
	}
}

func NewSelectorWithParameters(giniThreshold float64, topK int64) *Selector {
	if topK <= 0 {
		topK = 10
	}
	if giniThreshold <= 0.0 || giniThreshold >= 1.0 {
		giniThreshold = 0.5
	}

	return &Selector{
		giniThreshold: giniThreshold,
		topK:          topK,
	}
}

func computeGini(probs map[int64]float64) float64 {
	var sumSquares float64 = 0.0

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
	if gini > s.giniThreshold {
		return selectTopK(probs, int(s.topK))
	} else {
		return selectWeighted(probs)
	}
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
