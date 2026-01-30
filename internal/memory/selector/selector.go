package selector

// Selector chooses next song based on runtime graph.
type Selector interface {
	// TODO: Next(currentSongID) (songID, error)
	// Must NOT know about BaseGraph.
}
