package models

import "GO_player/internal/memory/basegraph"

type Album struct {
	ID        int64                `json:"id"`
	Title     string               `json:"title"`
	Songs     int64                `json:"id_songs"`
	BaseGraph *basegraph.BaseGraph `json:"-"`
}

func NewAlbum() *Album {
	return &Album{
		BaseGraph: basegraph.NewBaseGraph(),
	}
}
