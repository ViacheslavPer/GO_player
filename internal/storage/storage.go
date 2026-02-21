package storage

import (
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

func (db *DB) SetSong(songID int64, data []byte) error {
	key := fmt.Sprintf("song/%d", songID)
	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetSong(id int64) ([]byte, error) {
	key := fmt.Sprintf("song/%d", id)

	var res []byte

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		res = append(res, val...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) ListSongs() ([][]byte, error) {
	var res [][]byte

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
			res = append(res, val)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) SetAlbum(albumID int64, data []byte) error {
	key := fmt.Sprintf("album/%d", albumID)
	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetAlbum(id int64) ([]byte, error) {
	key := fmt.Sprintf("album/%d", id)

	var res []byte

	err := db.runTxnReadOnly(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		res = append(res, val...)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return res, nil
}

func (db *DB) ListAlbums() ([][]byte, error) {
	var res [][]byte

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

			res = append(res, val)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) SetBaseGraph(albumID int64, data []byte) error {
	key := fmt.Sprintf("graph/%d", albumID)

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetBaseGraph(albumID int64) ([]byte, error) {
	key := fmt.Sprintf("graph/%d", albumID)

	var res []byte

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

		res = append(res, val...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) SetPlaybackSession(data []byte) error {
	key := "session/playback"

	return db.runTxnReadWrite(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (db *DB) GetPlaybackSession() ([]byte, error) {
	key := "session/playback"

	var res []byte

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

		res = append(res, val...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) runTxnReadOnly(fn func(txn *badger.Txn) error) error {
	return db.badger.View(fn)
}

func (db *DB) runTxnReadWrite(fn func(txn *badger.Txn) error) error {
	return db.badger.Update(fn)
}
