package catalog

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/models"
	"GO_player/internal/playback"
	"GO_player/internal/storage"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"sync"
)

type Catalog interface {
	LoadBaseGraphEdges(albumID int64) (map[int64]map[int64]float64, error)
	LoadPlaybackSession() (*playback.PlaybackChain, error)
	LoadSong(songID int64) (*models.Song, error)
	LoadAlbum(albumID int64) (*models.Album, error)
	SaveBaseGraph(albumID int64, graph *basegraph.BaseGraph) error
	SavePlaybackSession(chain *playback.PlaybackChain) error
	SaveSong(songID int64, song *models.Song) error
	SaveAlbum(albumID int64, album *models.Album) error
	ListAlbums() ([]*models.Album, error)
	ListSongs() ([]*models.Song, error)
}

type catalogImpl struct {
	mu sync.Mutex
	db *storage.DB
}

func NewCatalog(db *storage.DB) Catalog {
	return &catalogImpl{db: db}
}

func (c *catalogImpl) LoadBaseGraphEdges(albumID int64) (map[int64]map[int64]float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.GetBaseGraph(albumID)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return map[int64]map[int64]float64{}, nil
	}

	var edges map[int64]map[int64]float64
	decoder := gob.NewDecoder(bytes.NewReader(val))
	if err := decoder.Decode(&edges); err != nil {
		return map[int64]map[int64]float64{}, nil
	}
	return edges, nil
}

func (c *catalogImpl) LoadPlaybackSession() (*playback.PlaybackChain, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.GetPlaybackSession()
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return &playback.PlaybackChain{}, nil
	}

	var pb playback.PlaybackChain
	if err := json.Unmarshal(val, &pb); err != nil {
		return &playback.PlaybackChain{}, nil
	}
	return &pb, nil
}

func (c *catalogImpl) LoadSong(songID int64) (*models.Song, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.GetSong(songID)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return &models.Song{}, nil
	}

	var song models.Song
	if err := json.Unmarshal(val, &song); err != nil {
		return &models.Song{}, nil
	}
	return &song, nil
}

func (c *catalogImpl) LoadAlbum(albumID int64) (*models.Album, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.GetAlbum(albumID)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return models.NewAlbum(), nil
	}

	album := models.NewAlbum()
	if err := json.Unmarshal(val, album); err != nil {
		return models.NewAlbum(), nil
	}
	return album, nil
}

func (c *catalogImpl) SaveBaseGraph(albumID int64, graph *basegraph.BaseGraph) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(graph.GetEdges()); err != nil {
		return err
	}
	return c.db.SetBaseGraph(albumID, buf.Bytes())
}

func (c *catalogImpl) SavePlaybackSession(chain *playback.PlaybackChain) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(chain)
	if err != nil {
		return err
	}
	return c.db.SetPlaybackSession(data)
}

func (c *catalogImpl) SaveSong(songID int64, song *models.Song) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(song)
	if err != nil {
		return err
	}
	return c.db.SetSong(songID, data)
}

func (c *catalogImpl) SaveAlbum(albumID int64, album *models.Album) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(album)
	if err != nil {
		return err
	}
	return c.db.SetAlbum(albumID, data)
}

func (c *catalogImpl) ListAlbums() ([]*models.Album, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.ListAlbums()
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return []*models.Album{}, nil
	}

	var albums []*models.Album
	for _, v := range val {
		album := models.NewAlbum()
		if err := json.Unmarshal(v, album); err != nil {
			continue
		}
		albums = append(albums, album)
	}
	return albums, nil
}

func (c *catalogImpl) ListSongs() ([]*models.Song, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, err := c.db.ListSongs()
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return []*models.Song{}, nil
	}

	var songs []*models.Song
	for _, v := range val {
		song := &models.Song{}
		if err := json.Unmarshal(v, song); err != nil {
			continue
		}
		songs = append(songs, song)
	}
	return songs, nil
}
