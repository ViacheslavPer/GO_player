package app

import (
	"GO_player/internal/catalog"
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/memory/runtime"
	"GO_player/internal/memory/selector"
	"GO_player/internal/models"
	"GO_player/internal/orchestrator"
	"GO_player/internal/storage"
	"context"
	"errors"
	"sync"
	"time"
)

type App struct {
	db                   *storage.DB
	catalog              catalog.Catalog
	albumID              int64
	orch                 *orchestrator.Orchestrator
	baseGraphRebuildChan <-chan bool
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   *sync.WaitGroup
}

func NewApp(dpPath string, albumID int64) (*App, error) {
	if dpPath == "" {
		return nil, errors.New("empty db path")
	}
	if albumID < 0 {
		return nil, errors.New("invalid album id")
	}

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

	pb, err := cat.LoadPlaybackSession()
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

	orch := orchestrator.NewOrchestrator(bg, rg, s, pb)
	bgChan := orch.GetBGRebuildChan()

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	app := &App{db: db, catalog: cat, albumID: albumID, orch: orch, baseGraphRebuildChan: bgChan, ctx: ctx, cancel: cancel, wg: wg}
	app.start()

	return app, nil
}

func (a *App) start() {
	a.wg.Add(2)
	go a.manageBaseGraphRebuild()
}

func (a *App) Stop() {
	//mu?
	a.cancel()
	a.wg.Wait()
}

func (a *App) manageBaseGraphRebuild() {
	defer a.wg.Done()
	for {
		select {
		case <-a.ctx.Done():
			return
		case _, ok := <-a.baseGraphRebuildChan:
			if !ok {
				return
			}
			if a.orch == nil {
				return //TODO: hadle error
			}
			bg := a.orch.GetBaseGraph()
			if bg == nil {
				return //TODO: hadle error
			}
			err := a.catalog.SaveBaseGraph(a.albumID, bg)
			if err != nil {
				return //TODO: hadle error
			}
		}
	}
}

func (a *App) manageRuntimeGraphTS() {
	defer a.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if a.orch == nil {
				return //TODO: hadle error
			}
			pb := a.orch.GetPlayBackChain()
			if pb == nil {
				return //TODO: hadle error
			}
			err := a.catalog.SavePlaybackSession(pb)
			if err != nil {
				return //TODO: hadle error
			}
		}
	}
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
	a.orch.ProcessFeedback(fromID, toID, listened, duration)
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
