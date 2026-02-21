// Package catalog — прослойка между storage (DB) и оркестратором/App.
// Даёт единый контракт для загрузки и сохранения состояния воспроизведения и каталога,
// не привязывая вызывающего к конкретной реализации хранилища.
package catalog

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/models"
	"GO_player/internal/playback"
	"GO_player/internal/storage"
)

// Catalog — интерфейс прослойки. Используется App для загрузки/сохранения состояния
// и получения списков для GUI. Реализация внутри ходит в storage.DB.
type Catalog interface {
	LoadBaseGraphEdges(albumID int64) (map[int64]map[int64]float64, error)
	LoadPlaybackSession() (playback.PlaybackChain, error)
	SaveBaseGraph(albumID int64, graph *basegraph.BaseGraph) error
	SavePlaybackSession(chain playback.PlaybackChain) error
	ListAlbums() ([]*models.Album, error)
	ListSongs() ([]*models.Song, error)
}

// catalogImpl — реализация Catalog поверх storage.DB. Логика — делегирование в db + комментарии.
type catalogImpl struct {
	db *storage.DB
}

// NewCatalog создаёт реализацию прослойки для переданного DB.
func NewCatalog(db *storage.DB) Catalog {
	return &catalogImpl{db: db}
}

// LoadBaseGraphEdges — рёбра графа для альбома из БД. App строит из них BaseGraph при старте.
func (c *catalogImpl) LoadBaseGraphEdges(albumID int64) (map[int64]map[int64]float64, error) {
	return c.db.GetBaseGraph(albumID)
}

// LoadPlaybackSession — последняя сохранённая цепочка воспроизведения. Если нет — пустая цепочка.
func (c *catalogImpl) LoadPlaybackSession() (playback.PlaybackChain, error) {
	// псевдокод: chain, err := c.db.GetPlaybackSession(); если ErrKeyNotFound — вернуть пустую цепочку и nil
	return c.db.GetPlaybackSession()
}

// SaveBaseGraph — сохранить граф альбома (после ребилда в оркестраторе).
func (c *catalogImpl) SaveBaseGraph(albumID int64, graph *basegraph.BaseGraph) error {
	return c.db.SetBaseGraph(albumID, graph)
}

// SavePlaybackSession — сохранить цепочку воспроизведения (после PlayNext/PlayBack в App).
func (c *catalogImpl) SavePlaybackSession(chain playback.PlaybackChain) error {
	return c.db.SetPlaybackSession(chain)
}

// ListAlbums — список альбомов для GUI.
func (c *catalogImpl) ListAlbums() ([]*models.Album, error) {
	return c.db.ListAlbums()
}

// ListSongs — список треков для GUI.
func (c *catalogImpl) ListSongs() ([]*models.Song, error) {
	return c.db.ListSongs()
}
