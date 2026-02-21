package storage

import (
	"GO_player/internal/memory/basegraph"
	"GO_player/internal/models"
	"GO_player/internal/playback"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

type DB struct {
	badger *badger.DB
}

func NewDB(path string) (*DB, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &DB{badger: db}, nil
}

func (db *DB) Close() error {
	return db.badger.Close()
}

func (db *DB) SetSong(song *models.Song) error {
	data, err := json.Marshal(song)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("song/%d", song.ID)

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetSong(id int64) (*models.Song, error) {
	key := fmt.Sprintf("song/%d", id)

	var song *models.Song

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		song = &models.Song{}
		return json.Unmarshal(val, song)
	})

	if err != nil {
		return nil, err
	}

	return song, nil
}

func (db *DB) ListSongs() ([]*models.Song, error) {
	var songs []*models.Song

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("song/")

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			song := &models.Song{}
			if err := json.Unmarshal(val, song); err != nil {
				return err
			}

			songs = append(songs, song)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return songs, nil
}

func (db *DB) SetAlbum(album *models.Album) error {
	albumData, err := json.Marshal(album)
	if err != nil {
		return err
	}

	albumKey := fmt.Sprintf("album/%d", album.ID)

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(albumKey), albumData); err != nil {
			return err
		}
		return nil
	})
}

func (db *DB) GetAlbum(id int64) (*models.Album, error) {
	albumKey := fmt.Sprintf("album/%d", id)

	album := models.NewAlbum()

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(albumKey))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		err = json.Unmarshal(val, album)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return album, nil
}

func (db *DB) ListAlbums() ([]*models.Album, error) {
	var albums []*models.Album

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("album/")

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			album := models.NewAlbum()
			if err := json.Unmarshal(val, album); err != nil {
				return err
			}

			albums = append(albums, album)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return albums, nil
}

func (db *DB) SetBaseGraph(albumID int64, graph *basegraph.BaseGraph) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(graph.GetEdges()); err != nil {
		return err
	}

	graphKey := fmt.Sprintf("graph/%d", albumID)

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(graphKey), buf.Bytes()); err != nil {
			return err
		}
		return nil
	})
}

func (db *DB) GetBaseGraph(albumID int64) (map[int64]map[int64]float64, error) {
	graphKey := fmt.Sprintf("graph/%d", albumID)
	edges := map[int64]map[int64]float64{}

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(graphKey))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		decoder := gob.NewDecoder(bytes.NewReader(val))
		if err := decoder.Decode(&edges); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return edges, nil
}

func (db *DB) SetPlaybackSession(playback playback.PlaybackChain) error {
	data, err := json.Marshal(playback)
	if err != nil {
		return err
	}

	key := "session/playback"

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetPlaybackSession() (playback.PlaybackChain, error) {
	key := "session/playback"

	var pb playback.PlaybackChain

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return json.Unmarshal(val, &pb)
	})

	if err != nil {
		return playback.PlaybackChain{}, err
	}

	return pb, nil
}

func (db *DB) runTxnReadOnly(fn func(txn *badger.Txn) error) error {
	return db.badger.View(fn)
}

func (db *DB) runTxnReadWrite(fn func(txn *badger.Txn) error) error {
	return db.badger.Update(fn)
}
