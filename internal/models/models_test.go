package models

import "testing"

func TestSong_ZeroValue(t *testing.T) {
	var s Song
	if s.ID != 0 {
		t.Errorf("zero Song ID should be 0, got %d", s.ID)
	}
	if s.Title != "" {
		t.Errorf("zero Song Title should be empty, got %q", s.Title)
	}
}

func TestSong_Fields(t *testing.T) {
	s := Song{ID: 1, Title: "test"}
	if s.ID != 1 {
		t.Errorf("Song.ID want 1, got %d", s.ID)
	}
	if s.Title != "test" {
		t.Errorf("Song.Title want %q, got %q", "test", s.Title)
	}
}
