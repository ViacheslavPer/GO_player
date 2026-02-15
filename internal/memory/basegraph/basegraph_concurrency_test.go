package basegraph

import (
	"sync"
	"testing"
)

// TestBaseGraph_ConcurrentReinforceAndRead verifies concurrent Reinforce and GetEdges/GetAllIDs do not race.
func TestBaseGraph_ConcurrentReinforceAndRead(t *testing.T) {
	const workers = 32
	const iterations = 500

	bg := NewBaseGraph()
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				fromID := int64((id + j) % 10)
				toID := int64(100 + (j % 5))
				bg.Reinforce(fromID, toID)
			}
		}(i)
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = bg.GetEdges(int64(j % 10))
				_ = bg.GetAllIDs()
			}
		}()
	}

	wg.Wait()

	ids := bg.GetAllIDs()
	if ids == nil {
		t.Fatal("GetAllIDs() returned nil after concurrent access")
	}
}

// TestBaseGraph_ConcurrentPenaltyAndReinforce verifies concurrent Penalty and Reinforce do not race.
func TestBaseGraph_ConcurrentPenaltyAndReinforce(t *testing.T) {
	bg := NewBaseGraph()
	for from := int64(0); from < 5; from++ {
		for to := int64(10); to < 15; to++ {
			bg.Reinforce(from, to)
		}
	}

	const workers = 20
	const iterations = 200
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				fromID := int64((id + j) % 5)
				toID := int64(10 + (j % 5))
				if j%2 == 0 {
					bg.Reinforce(fromID, toID)
				} else {
					bg.Penalty(fromID, toID)
				}
			}
		}(i)
	}

	wg.Wait()

	_ = bg.GetEdges(0)
	_ = bg.GetAllIDs()
}

// TestBaseGraph_EmptyGraphConcurrentReads verifies empty graph under concurrent reads.
func TestBaseGraph_EmptyGraphConcurrentReads(t *testing.T) {
	bg := NewBaseGraph()
	const workers = 50
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = bg.GetEdges(int64(j))
				_ = bg.GetAllIDs()
			}
		}()
	}
	wg.Wait()
}
