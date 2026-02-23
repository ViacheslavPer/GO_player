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

	db, err := storage.NewDB(dpPath, "", 0)
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
	a.wg.Add(1)
	go a.manageBaseGraphRebuild()
}

func (a *App) stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
}

func (a *App) Shutdown() error {
	a.stop()
	if a.orch != nil {
		a.orch.Shutdown()
	}
	if a.db != nil {
		return a.db.Shutdown()
	}
	return nil
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
				return //TODO: handle error
			}
			pb := a.orch.GetPlayBackChain()
			if pb == nil {
				return //TODO: handle error
			}
			err := a.catalog.SavePlaybackSession(pb)
			if err != nil {
				return //TODO: handle error
			}
		}
	}
}

func (a *App) PlayNext() (int64, bool) {
	if a.orch == nil {
		return 0, false
	}
	id, ok := a.orch.PlayNext()
	if ok {
		if pb := a.orch.GetPlayBackChain(); pb != nil {
			err := a.catalog.SavePlaybackSession(pb)
			if err != nil {
				return 0, false
				//TODO: handle error
			}
		}
	}
	return id, ok
}

func (a *App) PlayBack() (int64, bool) {
	if a.orch == nil {
		return 0, false
	}
	id, ok := a.orch.PlayBack()
	if ok {
		if pb := a.orch.GetPlayBackChain(); pb != nil {
			err := a.catalog.SavePlaybackSession(pb)
			if err != nil {
				return 0, false
				//TODO: handle
			}
		}
	}
	return id, ok
}

func (a *App) ProcessFeedback(fromID, toID int64, listened, duration float64) {
	if a.orch == nil {
		return
	}
	a.orch.ProcessFeedback(fromID, toID, listened, duration)
}

// for tests
func (a *App) ListSongs() ([]*models.Song, error) {
	return a.catalog.ListSongs()
}

func (a *App) ListAlbums() ([]*models.Album, error) {
	return a.catalog.ListAlbums()
}

func (a *App) Orchestrator() *orchestrator.Orchestrator {
	return a.orch
}
