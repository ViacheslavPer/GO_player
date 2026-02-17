package selector

import (
	"GO_player/internal/memory/runtime"
	"math"
	"testing"
)

func TestSelector_Next_EmptyGraph(t *testing.T) {
	s := NewSelector()
	rg := runtime.NewRuntimeGraph()

	if _, ok := s.Next(1, rg); ok {
		t.Fatalf("Next() on empty graph returned ok=true, want false")
	}
}

func TestSelector_Next_ReturnsOnlyExistingIDs(t *testing.T) {
	s := NewSelector()
	rg := runtime.NewRuntimeGraph()

	// High-gini-ish: many equal weights.
	want := map[int64]bool{}
	for i := int64(1); i <= 20; i++ {
		id := 100 + i
		want[id] = true
		rg.Reinforce(1, id, 1)
	}

	for i := 0; i < 200; i++ {
		id, ok := s.Next(1, rg)
		if !ok {
			t.Fatalf("Next() returned ok=false, want true")
		}
		if !want[id] {
			t.Fatalf("Next() returned id=%d not in expected set", id)
		}
	}
}

func TestSelector_Next_CoversMultipleDistributions(t *testing.T) {
	cases := []struct {
		name    string
		weights map[int64]float64
	}{
		{
			name: "gini around medium",
			weights: map[int64]float64{
				2: 7, 3: 2, 4: 1,
			},
		},
		{
			name: "gini around high",
			weights: map[int64]float64{
				2: 5, 3: 3, 4: 2,
			},
		},
		{
			name: "gini low (single dominant)",
			weights: map[int64]float64{
				2: 100, 3: 1, 4: 1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSelectorWithParameters(-1, 2, 0) // intentionally invalid; must still be usable
			rg := runtime.NewRuntimeGraph()
			want := map[int64]bool{}
			for id, w := range tc.weights {
				want[id] = true
				rg.Reinforce(1, id, w)
			}

			for i := 0; i < 50; i++ {
				got, ok := s.Next(1, rg)
				if !ok {
					t.Fatalf("Next() returned ok=false, want true")
				}
				if !want[got] {
					t.Fatalf("Next() returned id=%d not in expected set", got)
				}
			}
		})
	}
}

func TestNewSelectorWithParameters_ValidationBranches(t *testing.T) {
	// Exercise parameter validation branches without asserting specific defaults.
	_ = NewSelectorWithParameters(0.9, 0.1, 10)  // ok-ish
	_ = NewSelectorWithParameters(0.9, 0.1, -10) // topK <= 0
	_ = NewSelectorWithParameters(0.9, -1.0, 10) // giniLow invalid
	_ = NewSelectorWithParameters(-1.0, 0.2, 10) // giniHigh invalid
	_ = NewSelectorWithParameters(0.2, 0.5, 10)  // giniHigh <= giniLow
	_ = NewSelectorWithParameters(2.0, 2.0, 0)   // multiple invalids
}

func TestComputeTopK_ClampsToMinAndMax(t *testing.T) {
	// N=100 -> KMin=5, KMax=30 with current implementation.
	gotMin := computeTopK(100, -10.0)
	if gotMin != 5 {
		t.Fatalf("computeTopK(100, -10) = %d, want 5 (KMin)", gotMin)
	}

	gotMax := computeTopK(100, 10.0)
	if gotMax != 30 {
		t.Fatalf("computeTopK(100, 10) = %d, want 30 (KMax)", gotMax)
	}

	// Sanity: ratio in [0,1] should stay within [KMin,KMax].
	gotMid := computeTopK(100, 0.5)
	if gotMid < 5 || gotMid > 30 || math.IsNaN(float64(gotMid)) {
		t.Fatalf("computeTopK(100, 0.5) = %d, want within [5,30]", gotMid)
	}
}

func TestSelectWeighted_FallbackReturnPath(t *testing.T) {
	// Construct a degenerate distribution where the cumulative sum never exceeds f,
	// forcing the fallback return path.
	id, ok := selectWeighted(map[int64]float64{7: 0.0})
	if !ok || id != 7 {
		t.Fatalf("selectWeighted({7:0}) = (%d,%v), want (7,true)", id, ok)
	}
}
