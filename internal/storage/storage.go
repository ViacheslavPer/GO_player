package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
)

const defaultBackupInterval = 20 * time.Minute

type DB struct {
	badger *badger.DB
	backup *backupRunner
}

type backupRunner struct {
	badger     *badger.DB
	backupPath string
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	state      runState
}

type runState int

const (
	stateRunning runState = iota
	stateShutDown
)

func NewDB(path string, backupPath string, backupInterval time.Duration) (*DB, error) {
	opts := badger.DefaultOptions(path)
	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	if backupPath == "" {
		backupPath = path + ".backup"
	}
	if backupInterval <= 0 {
		backupInterval = defaultBackupInterval
	}

	runner := &backupRunner{
		badger:     badgerDB,
		backupPath: backupPath,
		interval:   backupInterval,
	}
	runner.start()

	return &DB{badger: badgerDB, backup: runner}, nil
}

func (db *DB) Close() error {
	return db.badger.Close()
}

// Shutdown stops the backup runner and closes the database. Call from app on exit.
func (db *DB) Shutdown() error {
	if db.backup != nil {
		db.backup.Shutdown()
	}
	return db.badger.Close()
}

func (b *backupRunner) start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == stateShutDown {
		return
	}
	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.wg.Add(1)
	go b.runBackupLoop()
}

func (b *backupRunner) stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
}

func (b *backupRunner) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == stateShutDown {
		return
	}
	b.stop()
	b.state = stateShutDown
}

func (b *backupRunner) runBackupLoop() {
	defer b.wg.Done()
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			_, _ = b.doBackup()
		}
	}
}

func (b *backupRunner) doBackup() (version uint64, err error) {
	f, err := os.Create(b.backupPath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	version, err = b.badger.Backup(f, 0)
	return version, err
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

const maxRetries = 3

func (db *DB) runTxnReadOnly(fn func(txn *badger.Txn) error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.badger.View(fn)
		if err == nil {
			return nil
		}
		if !errors.Is(err, badger.ErrConflict) {
			return err
		}
		if attempt == maxRetries-1 {
			return err
		}
	}
	return err
}

func (db *DB) runTxnReadWrite(fn func(txn *badger.Txn) error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.badger.Update(fn)
		if err == nil {
			return nil
		}
		if !errors.Is(err, badger.ErrConflict) {
			return err
		}
		if attempt == maxRetries-1 {
			return err
		}
	}
	return err
}
