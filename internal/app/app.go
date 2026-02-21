// Package app — точка входа и координация: создаёт DB, Catalog, Orchestrator,
// загружает состояние через Catalog, отдаёт GUI методы для управления воспроизведением.
package app

import (
	"GO_player/internal/catalog"
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
	"GO_player/internal/models"
	"GO_player/internal/orchestrator"
	"GO_player/internal/storage"
)

// App — держит хранилище, прослойку и оркестратор. GUI вызывает методы App, не трогая storage и orchestrator напрямую.
type App struct {
	db      *storage.DB
	catalog catalog.Catalog
	orch    *orchestrator.Orchestrator
	albumID int64 // какой альбом загружен (для SaveBaseGraph при ребилде)
}

// NewApp создаёт DB, Catalog, загружает состояние через Catalog и собирает Orchestrator.
// Псевдокод:
//
//	db := storage.NewDB(dpPath)
//	cat := catalog.NewCatalog(db)
//	edges := cat.LoadBaseGraphEdges(albumID)   // например 0
//	session := cat.LoadPlaybackSession()       // опционально
//	bg := basegraph.NewBaseGraph(); bg.SetEdges(edges)
//	rg := runtime.NewRuntimeGraph(); rg.BuildFromBase(bg)
//	s := selector.NewSelector()
//	orch := orchestrator.NewOrchestrator(bg, rg, s)
//	при необходимости: восстановить session в orch (если появится API)
//	return App{db, cat, orch, albumID}
func NewApp(dpPath string, albumID int64) (*App, error) {
	db, err := storage.NewDB(dpPath)
	if err != nil {
		return nil, err
	}

	cat := catalog.NewCatalog(db)

	edges, err := cat.LoadBaseGraphEdges(albumID)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	bg := basegraph.NewBaseGraph()
	if err := bg.SetEdges(edges); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := selector.NewSelector()
	rg := runtime.NewRuntimeGraph()
	rg.BuildFromBase(bg)

	orch := orchestrator.NewOrchestrator(bg, rg, s)

	// псевдокод: session, _ := cat.LoadPlaybackSession(); при наличии SetPlaybackChain(orch, session) — восстановить

	return &App{db: db, catalog: cat, orch: orch, albumID: albumID}, nil
}

// PlayNext — следующий трек. GUI вызывает после нажатия "вперёд".
// Псевдокод: id, ok := a.orch.PlayNext(); если ok — сохранить сессию через a.catalog.SavePlaybackSession(orch.PlaybackChain()); вернуть id, ok
func (a *App) PlayNext() (int64, bool) {
	id, ok := a.orch.PlayNext()
	// if ok { _ = a.catalog.SavePlaybackSession(*a.orch.PlaybackChain()) }
	_, _ = a.catalog, a.orch
	return id, ok
}

// PlayBack — предыдущий трек. GUI вызывает после нажатия "назад".
// Псевдокод: id, ok := a.orch.PlayBack(); если ok — сохранить сессию; вернуть id, ok
func (a *App) PlayBack() (int64, bool) {
	id, ok := a.orch.PlayBack()
	_, _ = a.catalog, a.orch
	return id, ok
}

// ProcessFeedback — фидбек по треку (дослушал/скип). Вызывать после окончания или смены трека.
// Псевдокод: a.orch.ProcessFeedbak(fromID, toID, listened, duration); при ребилде — a.catalog.SaveBaseGraph(a.albumID, a.orch.BaseGraph())
func (a *App) ProcessFeedback(fromID, toID int64, listened, duration float64) {
	a.orch.ProcessFeedbak(fromID, toID, listened, duration)
	// при ребилде (если оркестратор его сделал) — a.catalog.SaveBaseGraph(a.albumID, a.orch.BaseGraph())
	_, _, _ = a.catalog, a.albumID, a.orch
}

// ListSongs — список треков для GUI. Делегирует в Catalog.
func (a *App) ListSongs() ([]*models.Song, error) {
	return a.catalog.ListSongs()
}

// ListAlbums — список альбомов для GUI. Делегирует в Catalog.
func (a *App) ListAlbums() ([]*models.Album, error) {
	return a.catalog.ListAlbums()
}

// Orchestrator — доступ к оркестратору (для тестов или если GUI нужен прямой доступ).
func (a *App) Orchestrator() *orchestrator.Orchestrator {
	return a.orch
}

// Close — закрыть хранилище. Вызывать при выходе из приложения.
func (a *App) Close() error {
	return a.db.Close()
}
